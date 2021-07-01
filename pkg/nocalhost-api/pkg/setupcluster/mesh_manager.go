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
	"nocalhost/internal/nocalhost-api/model"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
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
	InitMeshDevSpace() error
	UpdateMeshDevSpace() error
	InjectMeshDevSpace() error
	GetBaseDevSpaceAppInfo() []MeshDevApp
	GetAPPInfo() ([]MeshDevApp, error)
}

type meshManager struct {
	mu          sync.Mutex
	client      *clientgo.GoClient
	cache       cache
	meshDevInfo MeshDevInfo
}

type MeshDevInfo struct {
	BaseNamespace    string       `json:"-"`
	MeshDevNamespace string       `json:"-"`
	Header           model.Header `json:"header"`
	APPS             []MeshDevApp `json:"apps"`
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

// TODO, use dynamicinformer to build cache
type cache struct {
	baseDevResources []unstructured.Unstructured
	meshDevResources []unstructured.Unstructured
}

func (c *cache) getResources() []unstructured.Unstructured {
	rs := make([]unstructured.Unstructured, len(c.baseDevResources))
	for i := range c.baseDevResources {
		c.baseDevResources[i].DeepCopyInto(&rs[i])
	}
	return rs
}

func (c *cache) getMeshDevResources() []unstructured.Unstructured {
	rs := make([]unstructured.Unstructured, len(c.meshDevResources))
	for i := range c.meshDevResources {
		c.meshDevResources[i].DeepCopyInto(&rs[i])
	}
	return rs
}

func (m *meshManager) InitMeshDevSpace() error {
	return m.initMeshDevSpace()
}

func (m *meshManager) UpdateMeshDevSpace() error {
	if err := m.setWorkloadStatus(); err != nil {
		return err
	}
	return m.InjectMeshDevSpace()
}

func (m *meshManager) InjectMeshDevSpace() error {
	// get dev space workloads from cache
	ws := make(map[string]int)
	for _, a := range m.meshDevInfo.APPS {
		for _, w := range a.Workloads {
			ws[w.Kind+"/"+w.Name] = w.Status
		}
	}
	irs := make([]unstructured.Unstructured, 0)
	drs := make([]unstructured.Unstructured, 0)
	for _, r := range m.cache.getResources() {
		if ws[r.GetKind()+"/"+r.GetName()] == ShouldBeInstalled {
			irs = append(irs, r)
			continue
		}
		if ws[r.GetKind()+"/"+r.GetName()] == ShouldBeDeleted {
			drs = append(drs, r)
		}
	}

	g, _ := errgroup.WithContext(context.Background())
	// apply workloads
	g.Go(func() error {
		return m.applyWorkloadsToMeshDevSpace(irs)
	})

	// update base dev space vs
	g.Go(func() error {
		return m.updateVirtualserviceOnBaseDevSpace(irs, drs)
	})

	// delete workloads
	g.Go(func() error {
		return m.deleteWorkloadsFromMeshDevSpace(drs)
	})
	return g.Wait()
}

func (m *meshManager) deleteWorkloadsFromMeshDevSpace(drs []unstructured.Unstructured) error {
	for _, r := range drs {
		log.Debugf("delete the workload %s/%s from %s", r.GetKind(), r.GetName(), m.meshDevInfo.MeshDevNamespace)
		if err := meshDevModify(m.meshDevInfo.MeshDevNamespace, &r); err != nil {
			return err
		}
		err := m.client.Delete(&r)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
		vs, err := genVirtualServiceForMeshDevSpace(m.meshDevInfo.BaseNamespace, r)
		if err != nil {
			log.Errorf("%+v", err)
			continue
		}
		log.Debugf("apply the Virtualservice/%s to the base namespace %s", r.GetName(), r.GetNamespace())
		if _, err := m.client.Apply(vs); err != nil {
			log.Errorf("%+v", err)
		}
	}
	return nil
}

func (m *meshManager) applyWorkloadsToMeshDevSpace(irs []unstructured.Unstructured) error {
	for _, r := range irs {
		log.Debugf("inject the workload %s/%s to %s", r.GetKind(), r.GetName(), m.meshDevInfo.MeshDevNamespace)
		if err := meshDevModify(m.meshDevInfo.MeshDevNamespace, &r); err != nil {
			return err
		}
		if _, err := m.client.Apply(&r); err != nil {
			return err
		}
		// delete vs form mesh dev space
		vs := &unstructured.Unstructured{}
		vs.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "VirtualService",
		})
		vs.SetNamespace(r.GetNamespace())
		vs.SetName(r.GetName())

		log.Debugf("delete the Virtualservice/%s from the base namespace %s", r.GetName(), r.GetNamespace())
		err := m.client.Delete(vs)
		if err != nil {
			log.Debug(err)
		}
	}
	return nil
}

func (m *meshManager) updateVirtualserviceOnBaseDevSpace(irs, drs []unstructured.Unstructured) error {
	// TODO, create vs by service name, not workload name
	// TODO, just update if the vs already exists
	info := m.meshDevInfo
	if info.Header.TraceKey != "" || info.Header.TraceValue != "" {
		for _, r := range irs {
			vs, err := genVirtualServiceForBaseDevSpace(
				info.BaseNamespace,
				info.MeshDevNamespace,
				r.GetName(),
				info.Header,
			)
			if err != nil {
				return err
			}
			log.Debugf("apply the Virtualservice/%s to the base namespace %s", r.GetName(), info.BaseNamespace)
			_, err = m.client.Apply(vs)
			if err != nil {
				return err
			}
		}
	}

	// TODO just delete the header match form vs
	for _, r := range drs {
		vs := &unstructured.Unstructured{}
		vs.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "networking.istio.io",
			Version: "v1alpha3",
			Kind:    "VirtualService",
		})
		vs.SetNamespace(r.GetNamespace())
		vs.SetName(r.GetName())

		log.Debugf("delete the Virtualservice/%s from the base namespace %s", r.GetName(), info.BaseNamespace)
		err := m.client.Delete(vs)
		if err != nil {
			log.Debug(err)
		}
	}
	return nil
}

func (m *meshManager) GetBaseDevSpaceAppInfo() []MeshDevApp {
	appNames := make([]string, 0)
	appInfo := make([]MeshDevApp, 0)
	appConfigsTmp := newAppMatcher(m.cache.getResources()).kind("Secret").namePrefix(appmeta.SecretNamePrefix).match()
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
		for _, r := range newAppMatcher(m.cache.getResources()).app(name).kind("Deployment").match() {
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
	for _, r := range newAppMatcher(m.cache.getResources()).excludeApps(appNames).kind("Deployment").match() {
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

func (m *meshManager) GetAPPInfo() ([]MeshDevApp, error) {
	if err := m.buildMeshDevCache(); err != nil {
		return nil, err
	}

	status := make(map[string]struct{})
	for _, r := range m.cache.getMeshDevResources() {
		status[r.GetKind()+"/"+r.GetName()] = struct{}{}
	}

	apps := m.GetBaseDevSpaceAppInfo()
	for i, a := range apps {
		for j, w := range a.Workloads {
			if _, ok := status[w.Kind+"/"+w.Name]; ok {
				apps[i].Workloads[j].Status = Selected
			}
		}
	}
	return apps, nil
}

func (m *meshManager) initMeshDevSpace() error {
	// apply app config
	appConfigsTmp := newAppMatcher(m.cache.getResources()).kind("Secret").namePrefix(appmeta.SecretNamePrefix).match()
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

		if err := meshDevModify(m.meshDevInfo.MeshDevNamespace, &c); err != nil {
			return err
		}
		_, err = m.client.Apply(&c)
		if err != nil {
			return err

		}
	}
	// get svc, gen vs
	svcs := newAppMatcher(m.cache.getResources()).kind("Service").match()
	vss := make([]v1alpha3.VirtualService, len(svcs))
	for i := range svcs {
		if err := meshDevModify(m.meshDevInfo.MeshDevNamespace, &svcs[i]); err != nil {
			return err
		}
		vs, err := genVirtualServiceForMeshDevSpace(m.meshDevInfo.BaseNamespace, svcs[i])
		if err != nil {
			return err
		}
		vss[i] = *vs
	}

	// apply svc and vs
	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error {
		for _, svc := range svcs {
			_, err := m.client.Apply(&svc)
			if err != nil {
				return err
			}
		}
		return nil
	})

	g.Go(func() error {
		for _, vs := range vss {
			_, err := m.client.Apply(&vs)
			if err != nil {
				return err
			}

		}
		return nil
	})

	return g.Wait()
}

func (m *meshManager) buildCache() error {
	rs := make([]schema.GroupVersionResource, 0)
	addGVR(&rs, schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	})
	addGVR(&rs, schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	})
	addGVR(&rs, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	})
	addGVR(&rs, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	})
	addGVR(&rs, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	})

	g, _ := errgroup.WithContext(context.Background())
	for _, r := range rs {
		r := r
		g.Go(func() error {
			l, err := m.client.DynamicClient.
				Resource(r).
				Namespace(m.meshDevInfo.BaseNamespace).
				List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return errors.WithStack(err)
			}
			m.mu.Lock()
			defer m.mu.Unlock()
			m.cache.baseDevResources = append(m.cache.baseDevResources, l.Items...)
			return nil
		})
	}
	return g.Wait()
}

func (m *meshManager) buildMeshDevCache() error {
	gvr := schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	}
	l, err := m.client.DynamicClient.
		Resource(gvr).
		Namespace(m.meshDevInfo.MeshDevNamespace).
		List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return errors.WithStack(err)
	}
	m.cache.meshDevResources = append(m.cache.meshDevResources, l.Items...)
	return nil
}

func (m *meshManager) getMeshDevSpaceWorkloads() ([]MeshDevWorkload, error) {
	if err := m.buildMeshDevCache(); err != nil {
		return nil, err
	}
	w := make([]MeshDevWorkload, 0)
	for _, r := range m.cache.getMeshDevResources() {
		w = append(w, MeshDevWorkload{
			Kind:   r.GetKind(),
			Name:   r.GetName(),
			Status: Installed,
		})
	}
	return w, nil
}

func (m *meshManager) setWorkloadStatus() error {
	log.Debug("set workloads status")
	devWs, err := m.getMeshDevSpaceWorkloads()
	if err != nil {
		return err
	}
	devMap := make(map[string]MeshDevWorkload)
	for _, w := range devWs {
		devMap[w.Kind+"/"+w.Name] = w
	}
	apps := m.meshDevInfo.APPS
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
	m.meshDevInfo.APPS = apps
	return nil

}

func addGVR(rs *[]schema.GroupVersionResource, gvr schema.GroupVersionResource) {
	*rs = append(*rs, gvr)
}

type appMatcher struct {
	matchResources []unstructured.Unstructured
}

func newAppMatcher(resources []unstructured.Unstructured) *appMatcher {
	m := &appMatcher{}
	m.matchResources = make([]unstructured.Unstructured, len(resources))
	for i := range resources {
		resources[i].DeepCopyInto(&m.matchResources[i])
	}
	return m
}

// match by kind
func (m *appMatcher) kind(kind string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	for _, r := range m.matchResources {
		if r.GetKind() == kind {
			match = append(match, r)
		}
	}
	m.matchResources = match
	return m
}

// match by app name
func (m *appMatcher) app(appName string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	for _, r := range m.matchResources {
		a := r.GetAnnotations()
		if a == nil {
			continue
		}
		if a[nocalhost.NocalhostApplicationName] == appName {
			match = append(match, r)
			continue
		}
		if a[nocalhost.HelmReleaseName] == appName {
			match = append(match, r)
		}
	}
	m.matchResources = match
	return m
}

// match exclude app name
func (m *appMatcher) excludeApp(appName string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	for _, r := range m.matchResources {
		a := r.GetAnnotations()
		if a == nil {
			match = append(match, r)
			continue
		}
		if a[nocalhost.NocalhostApplicationName] == appName {
			continue
		}
		if a[nocalhost.HelmReleaseName] == appName {
			continue
		}
		match = append(match, r)
	}
	m.matchResources = match
	return m
}

// match by app names
func (m *appMatcher) apps(appNames []string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	am := make(map[string]struct{})
	for _, n := range appNames {
		if n != "" {
			am[n] = struct{}{}
		}
	}
	for _, r := range m.matchResources {
		a := r.GetAnnotations()
		if a == nil {
			continue
		}
		if _, ok := am[a[nocalhost.NocalhostApplicationName]]; ok {
			match = append(match, r)
			continue
		}
		if _, ok := am[a[nocalhost.HelmReleaseName]]; ok {
			match = append(match, r)
		}
	}
	m.matchResources = match
	return m
}

// match exclude app names
func (m *appMatcher) excludeApps(appNames []string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	am := make(map[string]struct{})
	for _, n := range appNames {
		if n != "" {
			am[n] = struct{}{}
		}
	}
	for _, r := range m.matchResources {
		a := r.GetAnnotations()
		if a == nil {
			match = append(match, r)
			continue
		}
		if _, ok := am[a[nocalhost.NocalhostApplicationName]]; ok {
			continue
		}
		if _, ok := am[a[nocalhost.HelmReleaseName]]; ok {
			continue
		}
		match = append(match, r)
	}
	m.matchResources = match
	return m
}

func (m *appMatcher) name(name string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	for _, r := range m.matchResources {
		if r.GetName() == name {
			match = append(match, r)
		}
	}
	m.matchResources = match
	return m
}

func (m *appMatcher) namePrefix(prefix string) *appMatcher {
	match := make([]unstructured.Unstructured, 0)
	for _, r := range m.matchResources {
		if strings.HasPrefix(r.GetName(), prefix) {
			match = append(match, r)
		}
	}
	m.matchResources = match
	return m
}

func (m *appMatcher) names(name []string) *appMatcher {
	nm := make(map[string]struct{})
	for _, n := range name {
		nm[n] = struct{}{}
	}
	match := make([]unstructured.Unstructured, 0)
	for _, r := range m.matchResources {
		if _, ok := nm[r.GetName()]; ok {
			match = append(match, r)
		}
	}
	m.matchResources = match
	return m
}

func (m *appMatcher) match() []unstructured.Unstructured {
	return m.matchResources
}

func NewMeshManager(client *clientgo.GoClient, info MeshDevInfo) (MeshManager, error) {
	m := &meshManager{}
	m.client = client
	m.meshDevInfo = info
	// cache resources
	if err := m.buildCache(); err != nil {
		return nil, errors.WithStack(err)
	}
	return m, nil
}
