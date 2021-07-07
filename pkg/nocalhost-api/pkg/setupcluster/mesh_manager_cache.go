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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"strings"

	"nocalhost/internal/nhctl/nocalhost"
)

// TODO, use dynamicinformer to build cache
type cache struct {
	stopCh    chan struct{}
	informers dynamicinformer.DynamicSharedInformerFactory
}

func (c *cache) build() {
	rs := defaultGvr()
	for _, r := range rs {
		c.informers.ForResource(r)
	}
	c.informers.Start(c.stopCh)
	c.informers.WaitForCacheSync(c.stopCh)
}

func (c *cache) Configmap() informers.GenericInformer {
	return c.informers.ForResource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	})
}

func (c *cache) Service() informers.GenericInformer {
	return c.informers.ForResource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	})
}

func (c *cache) Secret() informers.GenericInformer {
	return c.informers.ForResource(schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	})
}

func (c *cache) VirtualService() informers.GenericInformer {
	return c.informers.ForResource(schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	})
}

func (c *cache) Deployment() informers.GenericInformer {
	return c.informers.ForResource(schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	})
}

func (c *cache) GetConfigmapsListByNameSpace(n string) []unstructured.Unstructured {
	return c.GetListByKindAndNamespace("Configmap", n)
}

func (c *cache) GetServicesListByNameSpace(n string) []unstructured.Unstructured {
	return c.GetListByKindAndNamespace("Service", n)
}

func (c *cache) GetVirtualServicesListByNameSpace(n string) []unstructured.Unstructured {
	return c.GetListByKindAndNamespace("VirtualServices", n)
}

func (c *cache) GetSecretsListByNameSpace(n string) []unstructured.Unstructured {
	return c.GetListByKindAndNamespace("Secret", n)
}

func (c *cache) GetDeploymentsListByNameSpace(n string) []unstructured.Unstructured {
	return c.GetListByKindAndNamespace("Deployment", n)
}

func (c *cache) GetListByKindAndNamespace(kind, n string) []unstructured.Unstructured {
	ret := make([]unstructured.Unstructured, 0)
	objs := make([]interface{}, 0)
	switch kind {
	case "Deployment":
		objs = c.Deployment().Informer().GetIndexer().List()
	case "Secret":
		objs = c.Secret().Informer().GetIndexer().List()
	case "Configmap":
		objs = c.Configmap().Informer().GetIndexer().List()
	case "Service":
		objs = c.Service().Informer().GetIndexer().List()
	case "VirtualService":
		objs = c.VirtualService().Informer().GetIndexer().List()
	}
	for _, obj := range objs {
		r := obj.(*unstructured.Unstructured)
		if r.GetNamespace() == n {
			ret = append(ret, *r.DeepCopy())
		}
	}
	return ret
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

func defaultGvr() []schema.GroupVersionResource {
	return []schema.GroupVersionResource{
		{
			Group:    "networking.istio.io",
			Version:  "v1alpha3",
			Resource: "virtualservices",
		},
		{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		},
		{
			Group:    "",
			Version:  "v1",
			Resource: "configmaps",
		},
		{
			Group:    "",
			Version:  "v1",
			Resource: "services",
		},
		{
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		},
	}
}
