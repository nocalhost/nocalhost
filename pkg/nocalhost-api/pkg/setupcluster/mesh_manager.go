/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package setupcluster

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic/dynamicinformer"

	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
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
	InjectMeshDevSpace(*MeshDevInfo) error
	DeleteTracingHeader(*MeshDevInfo) error
	GetBaseDevSpaceAppInfo(*MeshDevInfo) []MeshDevApp
	GetAPPInfo(*MeshDevInfo) ([]MeshDevApp, error)
	BuildCache() error
}

type meshManager struct {
	mu     sync.Mutex
	client *clientgo.GoClient
	cache  cache
	stopCh chan struct{}
}

type MeshDevInfo struct {
	BaseNamespace    string       `json:"-"`
	MeshDevNamespace string       `json:"-"`
	Header           model.Header `json:"header"`
	APPS             []MeshDevApp `json:"apps"`
	resources        meshDevResources
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

func (m *meshManager) InitMeshDevSpace(info *MeshDevInfo) error {
	return m.initMeshDevSpace(info)
}

func (m *meshManager) UpdateMeshDevSpace(info *MeshDevInfo) error {
	m.setWorkloadStatus(info)

	if err := m.InjectMeshDevSpace(info); err != nil {
		return err
	}

	return m.updateHeaderToVirtualServices(info)
}

func (m *meshManager) InjectMeshDevSpace(info *MeshDevInfo) error {
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

func (m *meshManager) DeleteTracingHeader(info *MeshDevInfo) error {

	for _, vs := range m.cache.GetVirtualServicesListByNameSpace(info.BaseNamespace) {
		ok, err := deleteHeaderFromVirtualService(&vs, info.MeshDevNamespace, info.Header)
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
	appConfigsTmp := newResourcesMatcher(m.cache.GetSecretsListByNameSpace(info.BaseNamespace)).
		namePrefix(appmeta.SecretNamePrefix).
		match()
	for _, c := range appConfigsTmp {
		name := c.GetName()[len(appmeta.SecretNamePrefix):]
		if name == nocalhost.DefaultNocalhostApplication {
			continue
		}

		val, found, err := unstructured.NestedString(c.UnstructuredContent(), "type")
		if !found || err != nil {
			continue
		}
		if val != appmeta.SecretType {
			continue
		}

		appNames = append(appNames, name)
		w := make([]MeshDevWorkload, 0)
		for _, r := range newResourcesMatcher(m.cache.GetDeploymentsListByNameSpace(info.BaseNamespace)).app(name).match() {
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
	for _, r := range newResourcesMatcher(m.cache.GetDeploymentsListByNameSpace(info.BaseNamespace)).
		excludeApps(appNames).
		match() {
		w = append(w, MeshDevWorkload{
			Kind: r.GetKind(),
			Name: r.GetName(),
		})
	}
	appInfo = append(appInfo, MeshDevApp{
		Name:      nocalhost.DefaultNocalhostApplication,
		Workloads: w,
	})

	return appInfo
}

func (m *meshManager) GetAPPInfo(info *MeshDevInfo) ([]MeshDevApp, error) {
	status := make(map[string]struct{})
	for _, r := range m.cache.GetDeploymentsListByNameSpace(info.MeshDevNamespace) {
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

func (m *meshManager) BuildCache() error {
	return m.buildCache()
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
			ok, err := deleteHeaderFromVirtualService(&v, info.MeshDevNamespace, info.Header)
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
	vs := make([]*unstructured.Unstructured, 0)
	for _, r := range info.resources.install {
		// update vs if already existed
		r := *r.DeepCopy()
		origVsMap := m.cache.MatchVirtualServiceByWorkload(r)

		for svcName, origVs := range origVsMap {
			for _, v := range origVs {
				if err := addHeaderToVirtualService(&v, info.MeshDevNamespace, svcName, info.Header); err != nil {
					return err
				}
				vs = append(vs, &v)
			}
		}
		if len(origVsMap) > 0 {
			continue
		}
		// generate vs if does not exist
		for _, s := range m.cache.MatchServicesByWorkload(r) {
			v, err := genVirtualServiceForBaseDevSpace(
				info.BaseNamespace,
				info.MeshDevNamespace,
				s.GetName(),
				info.Header,
			)
			if err != nil {
				return err
			}
			vs = append(vs, v)
		}

	}

	for i := range vs {
		log.Debugf("apply the VirtualService/%s to the base namespace %s", vs[i].GetName(), vs[i].GetNamespace())
		if _, err := m.client.ApplyForce(vs[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *meshManager) updateHeaderToVirtualServices(info *MeshDevInfo) error {
	header := info.Header
	if header.TraceKey == "" || header.TraceValue == "" {
		return nil
	}
	updatedVs := make([]unstructured.Unstructured, 0)
	for _, vs := range m.cache.GetVirtualServicesListByNameSpace(info.BaseNamespace) {
		isUpdate, err := updateHeaderToVirtualService(&vs, info.MeshDevNamespace, info.Header)
		if err != nil {
			return err
		}
		if isUpdate {
			updatedVs = append(updatedVs, vs)
		}
	}

	for i := range updatedVs {
		log.Debugf("apply the VirtualService/%s to the base namespace %s",
			updatedVs[i].GetName(), updatedVs[i].GetNamespace())
		if _, err := m.client.ApplyForce(&updatedVs[i]); err != nil {
			return err
		}
	}

	return nil
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
	appConfigsTmp := newResourcesMatcher(m.cache.GetSecretsListByNameSpace(info.BaseNamespace)).
		namePrefix(appmeta.SecretNamePrefix).
		match()
	for _, c := range appConfigsTmp {
		name := c.GetName()[len(appmeta.SecretNamePrefix):]
		if name == nocalhost.DefaultNocalhostApplication {
			continue
		}
		val, found, err := unstructured.NestedString(c.UnstructuredContent(), "type")
		if !found || err != nil {
			continue
		}
		if val != appmeta.SecretType {
			continue
		}

		if err := commonModifier(info.MeshDevNamespace, &c); err != nil {
			return err
		}
		log.Debugf("apply the %s/%s to dev namespace %s", c.GetKind(), c.GetName(), c.GetNamespace())
		_, err = m.client.ApplyForce(&c)
		if err != nil {
			return err

		}
	}
	// get svc, gen vs
	svcs := m.cache.GetServicesListByNameSpace(info.BaseNamespace)
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

func (m *meshManager) setInformerFactory() {
	m.cache.stopCh = make(chan struct{})
	m.cache.informers = dynamicinformer.NewDynamicSharedInformerFactory(m.client.DynamicClient, time.Minute)
}

func (m *meshManager) buildCache() error {
	m.cache.build()
	return nil
}

func (m *meshManager) getMeshDevSpaceWorkloads(info *MeshDevInfo) []MeshDevWorkload {
	w := make([]MeshDevWorkload, 0)
	for _, r := range m.cache.GetDeploymentsListByNameSpace(info.MeshDevNamespace) {
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
	apps := info.APPS
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
	info.APPS = apps
}

func (m *meshManager) tagResources(info *MeshDevInfo) {
	ws := make(map[string]int)
	for _, a := range info.APPS {
		for _, w := range a.Workloads {
			ws[w.Kind+"/"+w.Name] = w.Status
		}
	}
	irs := make([]unstructured.Unstructured, 0)
	drs := make([]unstructured.Unstructured, 0)
	for _, r := range m.cache.GetDeploymentsListByNameSpace(info.BaseNamespace) {
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
	m.setInformerFactory()
	return m
}
