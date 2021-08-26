/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package setupcluster

import (
	"context"
	"sort"
	"time"

	"golang.org/x/sync/errgroup"
	istiov1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

const (
	NotInstalled = iota
	ShouldBeInstalled
	Installed
	ShouldBeDeleted

	Unselected = NotInstalled
	Selected   = ShouldBeInstalled
)

type MeshManager interface {
	InitMeshDevSpace(*MeshDevInfo) error
	UpdateMeshDevSpace(*MeshDevInfo) error
	DeleteTracingHeader(*MeshDevInfo) error
	GetBaseDevSpaceAppInfo(*MeshDevInfo) []MeshDevApp
	GetAPPInfo(*MeshDevInfo) ([]MeshDevApp, error)
	Rollback(*MeshDevInfo) error
	GetMeshNamespaceNames() []string
	close()
}

type MeshDevInfo struct {
	BaseNamespace    string       `json:"-"`
	MeshDevNamespace string       `json:"namespace"`
	IsUpdateHeader   bool         `json:"-"`
	Header           model.Header `json:"header"`
	Apps             []MeshDevApp `json:"apps"`
	ReCreate         bool         `json:"-"`
	resources        meshDevResources
	rollback         rollback
}

type MeshDevApp struct {
	Name      string            `json:"name"`
	Workloads []MeshDevWorkload `json:"workloads"`
}

type MeshDevWorkload struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

type meshDevResources struct {
	install []unstructured.Unstructured
	delete  []unstructured.Unstructured
}

type rollback struct {
	header rollbackHeader
}

type rollbackHeader struct {
	add    map[string]*istiov1alpha3.HTTPRoute
	update map[string][]*istiov1alpha3.HTTPRoute
}

type SortAppsByName []MeshDevApp

func (a SortAppsByName) Len() int           { return len(a) }
func (a SortAppsByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortAppsByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

type SortWorkloadsByKindAndName []MeshDevWorkload

func (w SortWorkloadsByKindAndName) Len() int      { return len(w) }
func (w SortWorkloadsByKindAndName) Swap(i, j int) { w[i], w[j] = w[j], w[i] }
func (w SortWorkloadsByKindAndName) Less(i, j int) bool {
	return w[i].Kind+w[i].Name < w[j].Kind+w[j].Name
}

func (info *MeshDevInfo) SortApps() {
	sort.Sort(SortAppsByName(info.Apps))
	for i := 0; i < len(info.Apps); i++ {
		sort.Sort(SortWorkloadsByKindAndName(info.Apps[i].Workloads))
	}
}

func (info *MeshDevInfo) Validate() bool {
	return info.Header.TraceKey != "" &&
		info.Header.TraceValue != "" &&
		len(validation.IsDNS1123Label(info.MeshDevNamespace)) == 0
}

type meshManager struct {
	client *clientgo.GoClient
	cache  cache
}

func (m *meshManager) InitMeshDevSpace(info *MeshDevInfo) error {
	if err := m.initMeshDevSpace(info); err != nil {
		return err
	}
	if len(info.Apps) > 0 {
		return m.injectMeshDevSpace(info)
	}
	return nil
}

func (m *meshManager) UpdateMeshDevSpace(info *MeshDevInfo) error {
	m.setWorkloadStatus(info)

	if err := m.injectMeshDevSpace(info); err != nil {
		return err
	}

	if info.IsUpdateHeader {
		return m.updateHeaderToVirtualServices(info)
	}
	return nil
}

func (m *meshManager) DeleteTracingHeader(info *MeshDevInfo) error {
	for _, vs := range m.cache.GetVirtualServicesListByNamespace(info.BaseNamespace) {
		ok, err := deleteHeaderFromVirtualService(&vs, info)
		if err != nil {
			log.Error(err)
		}
		if ok {
			log.Debugf("delete the header %s:%s from VirtualService/%s, namespace %s",
				info.Header.TraceKey, info.Header.TraceValue, vs.GetName(), vs.GetNamespace())
			if _, err := m.client.ApplyForce(&vs); err != nil {
				log.Error(err)
			}
		}
	}
	return nil
}

func (m *meshManager) GetBaseDevSpaceAppInfo(info *MeshDevInfo) []MeshDevApp {
	appNames := make([]string, 0)
	appInfo := make([]MeshDevApp, 0)
	appConfigs := m.cache.GetAppConfigByNamespace(info.BaseNamespace)
	for _, c := range appConfigs {
		name := c.GetName()[len(appmeta.SecretNamePrefix):]
		if name == _const.DefaultNocalhostApplication {
			continue
		}

		appNames = append(appNames, name)
		w := make([]MeshDevWorkload, 0)
		for _, r := range m.cache.GetDeploymentsListByNamespaceAndAppName(info.BaseNamespace, name) {
			w = append(w, MeshDevWorkload{
				Kind: r.GetKind(),
				Name: r.GetName(),
			})
		}
		appInfo = append(appInfo, MeshDevApp{
			Name:      name,
			Workloads: w,
		})
	}

	// default.application
	w := make([]MeshDevWorkload, 0)
	for _, r := range newResourcesMatcher(m.cache.GetDeploymentsListByNamespace(info.BaseNamespace)).
		excludeApps(appNames).
		match() {
		w = append(w, MeshDevWorkload{
			Kind: r.GetKind(),
			Name: r.GetName(),
		})
	}
	appInfo = append(appInfo, MeshDevApp{
		Name:      _const.DefaultNocalhostApplication,
		Workloads: w,
	})

	return appInfo
}

func (m *meshManager) GetAPPInfo(info *MeshDevInfo) ([]MeshDevApp, error) {
	status := make(map[string]struct{})
	for _, r := range m.cache.GetDeploymentsListByNamespace(info.MeshDevNamespace) {
		status[r.GetKind()+"/"+r.GetName()] = struct{}{}
	}

	apps := m.GetBaseDevSpaceAppInfo(info)
	for i, a := range apps {
		for j, w := range a.Workloads {
			if _, ok := status[w.Kind+"/"+w.Name]; ok {
				apps[i].Workloads[j].Status = Selected
			}
		}
	}
	return apps, nil
}

func (m *meshManager) Rollback(info *MeshDevInfo) error {
	// rollback after add tracing header failure
	_ = wait.Poll(200*time.Millisecond, 5*time.Second, func() (bool, error) {
		for name, route := range info.rollback.header.add {
			r, err := m.cache.GetVirtualServiceByNamespaceAndName(info.BaseNamespace, name)
			if err != nil {
				log.Error(err)
				continue
			}
			r.SetManagedFields(nil)
			vs := &v1alpha3.VirtualService{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(r.UnstructuredContent(), vs); err != nil {
				log.Error(err)
				continue
			}
			routes := vs.Spec.Http
			for i := 0; i < len(routes); i++ {
				if routes[i].GetName() == route.Name {
					routes = routes[:i+copy(routes[i:], routes[i+1:])]
					i--
				}
			}
			vs.Spec.Http = routes
			if _, err := m.client.ApplyForce(vs); err != nil {
				log.Error(err)
				continue
			}
			log.Debugf("rollback %s/%s in namespace %s", r.GetKind(), r.GetName(), r.GetNamespace())
			delete(info.rollback.header.add, name)
		}
		if len(info.rollback.header.add) > 0 {
			return false, nil
		}
		return true, nil
	})

	// rollback after update tracing header failure
	_ = wait.Poll(200*time.Millisecond, 5*time.Second, func() (bool, error) {
		for name, routes := range info.rollback.header.update {
			routesMap := make(map[string]*istiov1alpha3.HTTPRoute)
			for _, route := range routes {
				routesMap[route.Name] = route
			}
			r, err := m.cache.GetVirtualServiceByNamespaceAndName(info.BaseNamespace, name)
			if err != nil {
				log.Error(err)
				continue
			}
			r.SetManagedFields(nil)
			vs := &v1alpha3.VirtualService{}
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(r.UnstructuredContent(), vs); err != nil {
				log.Error(err)
				continue
			}
			httpRoutes := vs.Spec.Http
			for i, route := range httpRoutes {
				if rollRoute, ok := routesMap[route.Name]; ok {
					httpRoutes[i] = rollRoute
				}
			}
			if _, err := m.client.ApplyForce(vs); err != nil {
				log.Error(err)
				continue
			}
			log.Debugf("rollback %s/%s in namespace %s", r.GetKind(), r.GetName(), r.GetNamespace())
			delete(info.rollback.header.update, name)
		}
		if len(info.rollback.header.update) > 0 {
			return false, nil
		}
		return true, nil
	})

	return nil
}

func (m *meshManager) GetMeshNamespaceNames() []string {
	label := map[string]string{
		"istio-injection":        "enabled",
		"nocalhost.dev/devspace": "base",
	}
	ns := m.cache.GetNamespaceListBySelector(labels.Set(label).AsSelector())
	ret := make([]string, len(ns))
	for i := range ns {
		ret[i] = ns[i].GetName()
	}
	return ret
}

func (m *meshManager) close() {
	m.cache.close()
}

func (m *meshManager) injectMeshDevSpace(info *MeshDevInfo) error {
	m.tagResources(info)

	log.Debugf("inject workloads into dev namespace %s", info.MeshDevNamespace)

	g, _ := errgroup.WithContext(context.Background())
	// apply workloads
	g.Go(func() error {
		return m.applyWorkloadsToMeshDevSpace(info)
	})

	// update base dev space vs
	g.Go(func() error {
		return m.updateVirtualServiceOnBaseDevSpace(info)
	})

	// delete workloads
	g.Go(func() error {
		return m.deleteWorkloadsFromMeshDevSpace(info)
	})
	return g.Wait()
}

func (m *meshManager) deleteWorkloadsFromMeshDevSpace(info *MeshDevInfo) error {
	for _, r := range info.resources.delete {
		r := *r.DeepCopy()
		log.Debugf("delete the workload %s/%s from %s", r.GetKind(), r.GetName(), info.MeshDevNamespace)
		if err := commonModifier(info.MeshDevNamespace, &r); err != nil {
			return err
		}
		err := m.client.Delete(&r)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
		vs, err := genVirtualServiceForMeshDevSpace(info.BaseNamespace, r)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
		log.Debugf("apply the VirtualService/%s to the base namespace %s", r.GetName(), r.GetNamespace())
		if _, err := m.client.ApplyForce(vs); err != nil {
			return err
		}
	}
	return nil
}

func (m *meshManager) applyWorkloadsToMeshDevSpace(info *MeshDevInfo) error {
	for _, r := range info.resources.install {
		r := *r.DeepCopy()
		log.Debugf("inject the workload %s/%s into dev namespace %s", r.GetKind(), r.GetName(), info.MeshDevNamespace)
		dependencies, err := meshDevModifier(info.MeshDevNamespace, &r)
		if err != nil {
			return err
		}

		if err := m.applyDependencyToMeshDevSpace(dependencies, info); err != nil {
			return err
		}

		if _, err := m.client.ApplyForce(&r); err != nil {
			return err
		}

		// get virtual service from mesh dev space by workload
		delVs := make([]unstructured.Unstructured, 0)
		for _, v := range m.cache.MatchVirtualServiceByWorkload(r) {
			delVs = append(delVs, v...)
		}
		// delete virtual service form mesh dev space
		for _, v := range delVs {
			log.Debugf("delete the VirtualService/%s from dev namespace %s", v.GetName(), v.GetNamespace())
			if err := m.client.Delete(&v); err != nil {
				log.Error(err)
			}
		}
	}
	return nil
}

func (m *meshManager) deleteHeaderFromVirtualService(info *MeshDevInfo) error {
	// delete header from vs
	vs := make([]*unstructured.Unstructured, 0)
	for _, r := range info.resources.delete {
		r := *r.DeepCopy()
		origVsMap := m.cache.MatchVirtualServiceByWorkload(r)
		origVs := make([]unstructured.Unstructured, 0)
		for _, ovs := range origVsMap {
			origVs = append(origVs, ovs...)
		}
		for _, v := range origVs {
			ok, err := deleteHeaderFromVirtualService(&v, info)
			if err != nil {
				return err
			}
			if ok {
				vs = append(vs, &v)
			}
		}
	}

	for i := range vs {
		log.Debugf("delete the header %s:%s from VirtualService/%s, namespace %s",
			info.Header.TraceKey, info.Header.TraceValue, vs[i].GetName(), vs[i].GetNamespace())
		if _, err := m.client.Apply(vs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *meshManager) addHeaderToVirtualService(info *MeshDevInfo) error {
	// update vs
	if info.Header.TraceKey == "" || info.Header.TraceValue == "" {
		log.Debugf("can not find tracing header to update virtual service in the namespace %s",
			info.BaseNamespace)
		return nil
	}

	for _, r := range info.resources.install {
		r := *r.DeepCopy()
		origVsMap := make(map[string][]unstructured.Unstructured)

		// update vs if already existed
		if err := wait.Poll(100*time.Millisecond, 8*time.Second, func() (bool, error) {
			origVsMap = m.cache.MatchVirtualServiceByWorkload(r)
			if len(origVsMap) == 0 {
				return true, nil
			}
			for svcName, origVs := range origVsMap {
				for _, v := range origVs {
					route, err := addHeaderToVirtualService(&v, svcName, info)
					if err != nil {
						return false, err
					}
					log.Debugf("apply the VirtualService/%s to the base namespace %s", v.GetName(), v.GetNamespace())
					if _, err := m.client.ApplyForce(&v); err != nil {
						if k8serrors.IsConflict(err) {
							log.Error(err)
							return false, nil
						}
						return false, err
					}
					if info.rollback.header.add == nil {
						info.rollback.header.add = make(map[string]*istiov1alpha3.HTTPRoute)
					}
					info.rollback.header.add[v.GetName()] = route
				}
			}
			return true, nil
		}); err != nil {
			return err
		}
		if len(origVsMap) > 0 {
			continue
		}

		// generate vs if does not exist
		for _, s := range m.cache.MatchServicesByWorkload(r) {
			v, route, err := genVirtualServiceForBaseDevSpace(
				info.BaseNamespace,
				info.MeshDevNamespace,
				s.GetName(),
				info.Header,
			)
			if err != nil {
				return err
			}
			log.Debugf("apply the VirtualService/%s to the base namespace %s", v.GetName(), v.GetNamespace())
			if _, err := m.client.ApplyForce(&v); err != nil {
				return err
			}
			if info.rollback.header.add == nil {
				info.rollback.header.add = make(map[string]*istiov1alpha3.HTTPRoute)
			}
			info.rollback.header.add[v.GetName()] = route
		}
	}

	return nil
}

func (m *meshManager) updateHeaderToVirtualServices(info *MeshDevInfo) error {
	header := info.Header
	if header.TraceKey == "" || header.TraceValue == "" {
		return nil
	}
	updatedVs := make(map[string]struct{})
	return wait.Poll(100*time.Millisecond, 8*time.Second, func() (bool, error) {
		for _, vs := range m.cache.GetVirtualServicesListByNamespace(info.BaseNamespace) {
			routes, isUpdate, err := updateHeaderToVirtualService(&vs, info)
			if err != nil {
				return false, err
			}
			_, ok := updatedVs[vs.GetName()]
			if isUpdate && !ok {
				log.Debugf("apply the VirtualService/%s to the base namespace %s",
					vs.GetName(), vs.GetNamespace())
				if _, err := m.client.ApplyForce(&vs); err != nil {
					if k8serrors.IsConflict(err) {
						log.Error(err)
						return false, nil
					}
					return false, err
				}

				if info.rollback.header.update == nil {
					info.rollback.header.update = make(map[string][]*istiov1alpha3.HTTPRoute)
				}
				info.rollback.header.update[vs.GetName()] = routes
				updatedVs[vs.GetName()] = struct{}{}
			}
		}
		return true, nil
	})
}

func (m *meshManager) updateVirtualServiceOnBaseDevSpace(info *MeshDevInfo) error {
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		return m.deleteHeaderFromVirtualService(info)
	})
	g.Go(func() error {
		return m.addHeaderToVirtualService(info)
	})
	return g.Wait()
}

func (m *meshManager) initMeshDevSpace(info *MeshDevInfo) error {
	// apply app config
	log.Debugf("init the dev namespace %s", info.MeshDevNamespace)
	appConfigs := m.cache.GetAppConfigByNamespace(info.BaseNamespace)
	for _, c := range appConfigs {
		name := c.GetName()[len(appmeta.SecretNamePrefix):]
		if name == _const.DefaultNocalhostApplication {
			continue
		}

		if err := commonModifier(info.MeshDevNamespace, &c); err != nil {
			return err
		}
		log.Debugf("apply the %s/%s to dev namespace %s", c.GetKind(), c.GetName(), c.GetNamespace())
		_, err := m.client.ApplyForce(&c)
		if err != nil {
			return err
		}
	}
	// get svc, gen vs
	svcs := m.cache.GetServicesListByNamespace(info.BaseNamespace)
	vss := make([]unstructured.Unstructured, len(svcs))
	for i := range svcs {
		if _, err := meshDevModifier(info.MeshDevNamespace, &svcs[i]); err != nil {
			return err
		}
		vs, err := genVirtualServiceForMeshDevSpace(info.BaseNamespace, svcs[i])
		if err != nil {
			return err
		}
		vss[i] = *vs
	}

	// apply svc and vs
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		for _, svc := range svcs {
			log.Debugf("apply the %s/%s to dev namespace %s", svc.GetKind(), svc.GetName(), svc.GetNamespace())
			_, err := m.client.ApplyForce(&svc)
			if err != nil {
				return err
			}
		}
		return nil
	})

	g.Go(func() error {
		for _, vs := range vss {
			log.Debugf("apply the %s/%s to dev namespace %s", vs.GetKind(), vs.GetName(), vs.GetNamespace())
			_, err := m.client.ApplyForce(&vs)
			if err != nil {
				return err
			}

		}
		return nil
	})

	return g.Wait()
}

func (m *meshManager) newCache() {
	m.cache = *newCache(m.client.DynamicClient)
}

func (m *meshManager) getMeshDevSpaceWorkloads(info *MeshDevInfo) []MeshDevWorkload {
	w := make([]MeshDevWorkload, 0)
	for _, r := range m.cache.GetDeploymentsListByNamespace(info.MeshDevNamespace) {
		w = append(w, MeshDevWorkload{
			Kind:   r.GetKind(),
			Name:   r.GetName(),
			Status: Installed,
		})
	}
	return w
}

func (m *meshManager) setWorkloadStatus(info *MeshDevInfo) {
	log.Debug("set workloads status")
	devWs := m.getMeshDevSpaceWorkloads(info)
	devMap := make(map[string]MeshDevWorkload)
	for _, w := range devWs {
		devMap[w.Kind+"/"+w.Name] = w
	}
	apps := info.Apps
	for i, a := range apps {
		for j, w := range a.Workloads {
			if w.Status == Selected && devMap[w.Kind+"/"+w.Name].Status == Installed {
				apps[i].Workloads[j].Status = Installed
			}
			if w.Status == Unselected && devMap[w.Kind+"/"+w.Name].Status == Installed {
				apps[i].Workloads[j].Status = ShouldBeDeleted
			}
		}
	}
	info.Apps = apps
}

func (m *meshManager) tagResources(info *MeshDevInfo) {
	ws := make(map[string]int)
	for _, a := range info.Apps {
		for _, w := range a.Workloads {
			ws[w.Kind+"/"+w.Name] = w.Status
		}
	}
	irs := make([]unstructured.Unstructured, 0)
	drs := make([]unstructured.Unstructured, 0)
	for _, r := range m.cache.GetDeploymentsListByNamespace(info.BaseNamespace) {
		if ws[r.GetKind()+"/"+r.GetName()] == ShouldBeInstalled {
			irs = append(irs, r)
			continue
		}
		if ws[r.GetKind()+"/"+r.GetName()] == ShouldBeDeleted {
			drs = append(drs, r)
		}
	}

	info.resources.install = irs
	info.resources.delete = drs
}

func (m *meshManager) applyDependencyToMeshDevSpace(dependencies []MeshDevWorkload, info *MeshDevInfo) error {
	wm := make(map[string][]string)
	for _, d := range dependencies {
		if _, ok := wm[d.Kind]; ok {
			wm[d.Kind] = append(wm[d.Kind], d.Name)
			continue
		}
		wm[d.Kind] = []string{d.Name}
	}

	// get resources from cache
	rs := make([]unstructured.Unstructured, 0)
	for k, v := range wm {
		rs = append(rs, newResourcesMatcher(m.cache.GetListByKindAndNamespace(k, info.BaseNamespace)).
			names(v).match()...)
	}

	// apply resources
	for _, r := range rs {
		log.Debugf("inject the workload dependency %s/%s into dev namespace %s", r.GetKind(), r.GetName(), info.MeshDevNamespace)
		if err := commonModifier(info.MeshDevNamespace, &r); err != nil {
			return err
		}
		if _, err := m.client.ApplyForce(&r); err != nil {
			return err
		}
	}
	return nil
}

func NewMeshManager(client *clientgo.GoClient) MeshManager {
	m := &meshManager{}
	m.client = client
	m.newCache()
	return m
}
