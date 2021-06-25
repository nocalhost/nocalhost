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
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

type MeshManager interface {
	InitMeshDevSpace() error
	UpdateDstDevSpace() error
	InjectMeshDevSpace() error
	GetBaseDevSpaceAppInfo() []model.MeshDevApp
}

type meshManager struct {
	mu          sync.Mutex
	client      *clientgo.GoClient
	cache       cache
	meshDevInfo model.MeshDevInfo
}

type cache struct {
	resources []unstructured.Unstructured
}

func (c *cache) getResources() []unstructured.Unstructured {
	rs := make([]unstructured.Unstructured, len(c.resources))
	for i := range c.resources {
		c.resources[i].DeepCopyInto(&rs[i])
	}
	return rs
}

func (m *meshManager) InitMeshDevSpace() error {
	if err := m.initMeshDevSpace(); err != nil {
		return err
	}
	if len(m.meshDevInfo.Header) > 0 {
		return m.updateVirtualserviceOnBaseDevSpace()
	}
	return nil
}

func (m *meshManager) UpdateDstDevSpace() error {
	return m.initMeshDevSpace()
}

func (m *meshManager) InjectMeshDevSpace() error {
	// get dev space workloads from cache
	ws := make(map[string]struct{})
	for _, a := range m.meshDevInfo.APPS {
		for _, w := range a.Workloads {
			ws[w.Kind+"/"+w.Name] = struct{}{}
		}
	}
	rs := make([]unstructured.Unstructured, 0)
	for _, r := range m.cache.getResources() {
		if _, ok := ws[r.GetKind()+"/"+r.GetName()]; ok {
			if err := meshDevModify(m.meshDevInfo.MeshDevNamespace, &r); err != nil {
				return err
			}
			rs = append(rs, r)
		}
	}

	for i := range rs {
		if _, err := m.client.Apply(&rs[i]); err != nil {
			return err
		}
	}

	return nil
}

func (m *meshManager) updateVirtualserviceOnBaseDevSpace() error {
	// TODO, create vs by service name, not workload name
	// TODO, just update if the vs already exists
	ws := make([]model.MeshDevWorkload, 0)
	for _, a := range m.meshDevInfo.APPS {
		ws = append(ws, a.Workloads...)
	}

	for _, w := range ws {
		vs, err := genVirtualServiceForBaseDevSpace(
			m.meshDevInfo.BaseNamespace,
			m.meshDevInfo.MeshDevNamespace,
			w.Name,
			m.meshDevInfo.Header,
		)
		if err != nil {
			return err
		}
		_, err = m.client.Apply(vs)
		if err != nil {
			return err
		}
		log.Debugf("apply the virtualservice: %s/%s", m.meshDevInfo.BaseNamespace, w.Name)
	}
	return nil
}

func (m *meshManager) GetBaseDevSpaceAppInfo() []model.MeshDevApp {
	appNames := make([]string, 0)
	appInfo := make([]model.MeshDevApp, 0)
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
		w := make([]model.MeshDevWorkload, 0)
		for _, r := range newAppMatcher(m.cache.getResources()).app(name).kind("Deployment").match() {
			w = append(w, model.MeshDevWorkload{
				Kind: r.GetKind(),
				Name: r.GetName(),
			})
		}
		appInfo = append(appInfo, model.MeshDevApp{
			Name:      name,
			Workloads: w,
		})
	}

	// default.application
	w := make([]model.MeshDevWorkload, 0)
	for _, r := range newAppMatcher(m.cache.getResources()).excludeApps(appNames).kind("Deployment").match() {
		w = append(w, model.MeshDevWorkload{
			Kind: r.GetKind(),
			Name: r.GetName(),
		})
	}
	appInfo = append(appInfo, model.MeshDevApp{
		Name:      nocalhost.DefaultNocalhostApplication,
		Workloads: w,
	})

	return appInfo
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

func (m *meshManager) setBaseDevSpacePatchResources() error {
	//TODO
	return nil
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
			m.cache.resources = append(m.cache.resources, l.Items...)
			return nil
		})

	}
	return g.Wait()
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

func NewMeshManager(client *clientgo.GoClient, info model.MeshDevInfo) (MeshManager, error) {
	m := &meshManager{}
	m.client = client
	m.meshDevInfo = info
	// cache resources
	if err := m.buildCache(); err != nil {
		return nil, errors.WithStack(err)
	}
	return m, nil
}
