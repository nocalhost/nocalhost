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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	toolscache "k8s.io/client-go/tools/cache"

	"nocalhost/internal/nhctl/nocalhost"
)

const (
	ConfigMap      = "ConfigMap"
	Service        = "Service"
	VirtualService = "VirtualService"
	Secret         = "Secret"
	Deployment     = "Deployment"
)

type ExtendInformer interface {
	informers.GenericInformer
	ByIndex(indexName, indexedValue string) []unstructured.Unstructured
}

type Informer struct {
	informers.GenericInformer
}

func (informer *Informer) ByIndex(indexName, indexedValue string) []unstructured.Unstructured {
	objs, _ := informer.Informer().GetIndexer().ByIndex(indexName, indexedValue)
	ret := make([]unstructured.Unstructured, len(objs))
	for i := range objs {
		ret[i] = *objs[i].(*unstructured.Unstructured).DeepCopy()
	}
	return ret
}

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

func (c *cache) ConfigMap() ExtendInformer {
	return &Informer{
		GenericInformer: c.informers.ForResource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "configmaps",
		}),
	}
}

func (c *cache) Service() ExtendInformer {
	return &Informer{
		GenericInformer: c.informers.ForResource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "services",
		}),
	}
}

func (c *cache) Secret() ExtendInformer {
	return &Informer{
		GenericInformer: c.informers.ForResource(schema.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		}),
	}
}

func (c *cache) VirtualService() ExtendInformer {
	return &Informer{
		GenericInformer: c.informers.ForResource(schema.GroupVersionResource{
			Group:    "networking.istio.io",
			Version:  "v1alpha3",
			Resource: "virtualservices",
		}),
	}
}

func (c *cache) Deployment() ExtendInformer {
	return &Informer{
		GenericInformer: c.informers.ForResource(schema.GroupVersionResource{
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		}),
	}
}

func (c *cache) GetConfigMapsListByNamespace(ns string) []unstructured.Unstructured {
	return c.ConfigMap().ByIndex(toolscache.NamespaceIndex, ns)
}

func (c *cache) GetServicesListByNamespace(ns string) []unstructured.Unstructured {
	return c.Service().ByIndex(toolscache.NamespaceIndex, ns)
}

func (c *cache) GetVirtualServicesListByNamespace(ns string) []unstructured.Unstructured {
	return c.VirtualService().ByIndex(toolscache.NamespaceIndex, ns)
}

func (c *cache) GetSecretsListByNamespace(ns string) []unstructured.Unstructured {
	return c.Secret().ByIndex(toolscache.NamespaceIndex, ns)
}

func (c *cache) GetDeploymentsListByNamespace(ns string) []unstructured.Unstructured {
	return c.Deployment().ByIndex(toolscache.NamespaceIndex, ns)
}

func (c *cache) GetListByKindAndNamespace(kind, ns string) []unstructured.Unstructured {
	switch kind {
	case Deployment:
		return c.Deployment().ByIndex(toolscache.NamespaceIndex, ns)
	case Secret:
		return c.Secret().ByIndex(toolscache.NamespaceIndex, ns)
	case ConfigMap:
		return c.ConfigMap().ByIndex(toolscache.NamespaceIndex, ns)
	case Service:
		return c.Service().ByIndex(toolscache.NamespaceIndex, ns)
	case VirtualService:
		return c.VirtualService().ByIndex(toolscache.NamespaceIndex, ns)
	}
	return []unstructured.Unstructured{}
}

func (c *cache) MatchServicesByWorkload(r unstructured.Unstructured) []unstructured.Unstructured {
	ns := r.GetNamespace()
	if ns == corev1.NamespaceAll {
		return make([]unstructured.Unstructured, 0)
	}
	ls, _, _ := unstructured.NestedStringMap(r.UnstructuredContent(), "spec", "template", "metadata", "labels")

	return resourcesFilter(c.GetServicesListByNamespace(ns), func(r unstructured.Unstructured) bool {
		m, _, _ := unstructured.NestedStringMap(r.UnstructuredContent(), "spec", "selector")
		return labels.Set(m).AsSelector().Matches(labels.Set(ls))
	})
}

func (c *cache) MatchVirtualServiceByWorkload(r unstructured.Unstructured) map[string][]unstructured.Unstructured {
	vmap := make(map[string][]unstructured.Unstructured)
	ns := r.GetNamespace()
	if ns == corev1.NamespaceAll {
		return vmap
	}

	smap := make(map[string]struct{})
	svc := c.MatchServicesByWorkload(r)
	for _, s := range svc {
		smap[s.GetName()] = struct{}{}
	}

	vs := c.GetVirtualServicesListByNamespace(ns)
	for _, v := range vs {
		hosts, _, _ := unstructured.NestedStringSlice(v.UnstructuredContent(), "spec", "hosts")
		for _, host := range hosts {
			if _, ok := smap[host]; !ok {
				continue
			}
			if _, ok := vmap[host]; ok {
				vmap[host] = append(vmap[host], v)
				continue
			}
			vmap[host] = []unstructured.Unstructured{v}
		}
	}
	return vmap
}

type resourcesMatcher struct {
	resources []unstructured.Unstructured
}

func newResourcesMatcher(resources []unstructured.Unstructured) *resourcesMatcher {
	m := &resourcesMatcher{}
	m.resources = make([]unstructured.Unstructured, len(resources))
	for i := range resources {
		resources[i].DeepCopyInto(&m.resources[i])
	}
	return m
}

// match by kind
func (m *resourcesMatcher) kind(kind string) *resourcesMatcher {
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		return r.GetKind() == kind
	})
	return m
}

// match by app name
func (m *resourcesMatcher) app(appName string) *resourcesMatcher {
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		a := r.GetAnnotations()
		if a == nil {
			return false
		}
		if a[nocalhost.NocalhostApplicationName] == appName {
			return true
		}
		if a[nocalhost.HelmReleaseName] == appName {
			return true
		}
		return false
	})
	return m
}

// match exclude app name
func (m *resourcesMatcher) excludeApp(appName string) *resourcesMatcher {
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		a := r.GetAnnotations()
		if a == nil {
			return true
		}
		if a[nocalhost.NocalhostApplicationName] == appName {
			return false
		}
		if a[nocalhost.HelmReleaseName] == appName {
			return false
		}
		return true
	})
	return m
}

// match by app names
func (m *resourcesMatcher) apps(appNames []string) *resourcesMatcher {
	am := make(map[string]struct{})
	for _, n := range appNames {
		if n != "" {
			am[n] = struct{}{}
		}
	}
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		a := r.GetAnnotations()
		if a == nil {
			return false
		}
		if _, ok := am[a[nocalhost.NocalhostApplicationName]]; ok {
			return true
		}
		if _, ok := am[a[nocalhost.HelmReleaseName]]; ok {
			return true
		}
		return false
	})
	return m
}

// match exclude app names
func (m *resourcesMatcher) excludeApps(appNames []string) *resourcesMatcher {
	am := make(map[string]struct{})
	for _, n := range appNames {
		if n != "" {
			am[n] = struct{}{}
		}
	}
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		a := r.GetAnnotations()
		if a == nil {
			return true
		}
		if _, ok := am[a[nocalhost.NocalhostApplicationName]]; ok {
			return false
		}
		if _, ok := am[a[nocalhost.HelmReleaseName]]; ok {
			return false
		}
		return true
	})
	return m
}

func (m *resourcesMatcher) name(name string) *resourcesMatcher {
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		return r.GetName() == name
	})
	return m
}

func (m *resourcesMatcher) namePrefix(prefix string) *resourcesMatcher {
	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		return strings.HasPrefix(r.GetName(), prefix)
	})
	return m
}

func (m *resourcesMatcher) names(name []string) *resourcesMatcher {
	nm := make(map[string]struct{})
	for _, n := range name {
		nm[n] = struct{}{}
	}

	m.resources = resourcesFilter(m.resources, func(r unstructured.Unstructured) bool {
		_, ok := nm[r.GetName()]
		return ok
	})

	return m
}

func (m *resourcesMatcher) match() []unstructured.Unstructured {
	return m.resources
}

func resourcesFilter(rs []unstructured.Unstructured, f func(
	r unstructured.Unstructured) bool) []unstructured.Unstructured {
	ret := make([]unstructured.Unstructured, 0)
	for _, r := range rs {
		if f(r) {
			ret = append(ret, r)
		}
	}
	return ret
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
