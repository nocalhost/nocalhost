/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package setupcluster

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	toolscache "k8s.io/client-go/tools/cache"

	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/const"
)

const (
	ConfigMap      = "ConfigMap"
	Service        = "Service"
	VirtualService = "VirtualService"
	Secret         = "Secret"
	Deployment     = "Deployment"

	ApplicationIndex       = "nocalhostApplication"
	ApplicationConfigIndex = "nocalhostApplicationConfig"

	ClusterInformerFactory = "ClusterInformerFactory"

	defaultResync  = 10 * time.Minute
	defaultLRUSize = 12
)

type ExtendInformer interface {
	informers.GenericInformer
	ByIndex(indexName, indexedValue string) []unstructured.Unstructured
	GetList() []unstructured.Unstructured
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

func (informer *Informer) GetList() []unstructured.Unstructured {
	objs := informer.Informer().GetIndexer().List()
	ret := make([]unstructured.Unstructured, len(objs))
	for i := range objs {
		ret[i] = *objs[i].(*unstructured.Unstructured).DeepCopy()
	}
	return ret
}

func IndexByAppName(obj interface{}) ([]string, error) {
	r, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return []string{}, nil
	}

	annot := r.GetAnnotations()

	if len(annot) == 0 {
		return []string{}, nil
	}
	if name := annot[_const.NocalhostApplicationName]; name != "" {
		return []string{fmt.Sprintf("%s/%s", r.GetNamespace(), name)}, nil
	}
	if name := annot[_const.HelmReleaseName]; name != "" {
		return []string{fmt.Sprintf("%s/%s", r.GetNamespace(), name)}, nil
	}

	return []string{}, nil
}

func IndexAppConfig(obj interface{}) ([]string, error) {
	r, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return []string{}, nil
	}
	if r.GetKind() != Secret {
		return []string{}, nil
	}
	if !strings.HasPrefix(r.GetName(), appmeta.SecretNamePrefix) {
		return []string{}, nil
	}

	val, found, err := unstructured.NestedString(r.UnstructuredContent(), "type")
	if !found || err != nil {
		return []string{}, nil
	}
	if val != appmeta.SecretType {
		return []string{}, nil
	}

	return []string{r.GetNamespace()}, nil
}

type informerFactory struct {
	factory dynamicinformer.DynamicSharedInformerFactory
	stopCh  chan struct{}
	started map[schema.GroupVersionResource]bool
}

func (f *informerFactory) close() {
	close(f.stopCh)
}

type cache struct {
	mu     sync.Mutex
	client dynamic.Interface
	lru    *lru.Cache
}

func (c *cache) getInformerFactory(ns string) *informerFactory {
	c.mu.Lock()
	defer c.mu.Unlock()

	factory, exist := c.lru.Get(ns)
	if exist {
		return factory.(*informerFactory)
	}

	f := &informerFactory{
		factory: dynamicinformer.NewFilteredDynamicSharedInformerFactory(
			c.client, defaultResync, ns, nil),
		stopCh:  make(chan struct{}),
		started: make(map[schema.GroupVersionResource]bool),
	}
	if ns == metav1.NamespaceAll {
		c.lru.Add(ClusterInformerFactory, f)
	} else {
		c.lru.Add(ns, f)
	}
	return f
}

func (c *cache) getInformer(ns string, gvr schema.GroupVersionResource) ExtendInformer {
	f := c.getInformerFactory(ns)
	informer := f.factory.ForResource(gvr)
	_ = informer.Informer().AddIndexers(toolscache.Indexers{ApplicationIndex: IndexByAppName})
	if gvr.Resource == "secrets" {
		_ = informer.Informer().AddIndexers(toolscache.Indexers{ApplicationConfigIndex: IndexAppConfig})
	}
	if !f.started[gvr] {
		go informer.Informer().Run(f.stopCh)
		toolscache.WaitForCacheSync(f.stopCh, informer.Informer().HasSynced)
		f.started[gvr] = true
	}

	return &Informer{informer}
}

func (c *cache) close() {
	c.lru.Purge()
}

func newCache(client dynamic.Interface) *cache {
	lruCache, _ := lru.NewWithEvict(defaultLRUSize, func(key interface{}, value interface{}) {
		value.(*informerFactory).close()
	})
	return &cache{
		lru:    lruCache,
		client: client,
	}
}

func (c *cache) ConfigMap(ns string) ExtendInformer {
	return c.getInformer(ns, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	})
}

func (c *cache) Service(ns string) ExtendInformer {
	return c.getInformer(ns, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "services",
	})
}

func (c *cache) Secret(ns string) ExtendInformer {
	return c.getInformer(ns, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	})
}

func (c *cache) Namespace() ExtendInformer {
	return c.getInformer(metav1.NamespaceAll, schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	})
}

func (c *cache) VirtualService(ns string) ExtendInformer {
	return c.getInformer(ns, schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "virtualservices",
	})
}

func (c *cache) Deployment(ns string) ExtendInformer {
	return c.getInformer(ns, schema.GroupVersionResource{
		Group:    "apps",
		Version:  "v1",
		Resource: "deployments",
	})
}

func (c *cache) GetConfigMapsListByNamespace(ns string) []unstructured.Unstructured {
	return c.ConfigMap(ns).GetList()
}

func (c *cache) GetServicesListByNamespace(ns string) []unstructured.Unstructured {
	return c.Service(ns).GetList()
}

func (c *cache) GetVirtualServicesListByNamespace(ns string) []unstructured.Unstructured {
	return c.VirtualService(ns).GetList()
}

func (c *cache) GetSecretsListByNamespace(ns string) []unstructured.Unstructured {
	return c.Secret(ns).GetList()
}

func (c *cache) GetDeploymentsListByNamespace(ns string) []unstructured.Unstructured {
	return c.Deployment(ns).GetList()
}

func (c *cache) GetConfigMapsListByNamespaceAndAppName(ns, appName string) []unstructured.Unstructured {
	return c.ConfigMap(ns).ByIndex(ApplicationIndex, fmt.Sprintf("%s/%s", ns, appName))
}

func (c *cache) GetServicesListByNamespaceAndAppName(ns, appName string) []unstructured.Unstructured {
	return c.Service(ns).ByIndex(ApplicationIndex, fmt.Sprintf("%s/%s", ns, appName))
}

func (c *cache) GetVirtualServicesListByNamespaceAndAppName(ns, appName string) []unstructured.Unstructured {
	return c.VirtualService(ns).ByIndex(ApplicationIndex, fmt.Sprintf("%s/%s", ns, appName))
}

func (c *cache) GetSecretsListByNamespaceAndAppName(ns, appName string) []unstructured.Unstructured {
	return c.Secret(ns).ByIndex(ApplicationIndex, fmt.Sprintf("%s/%s", ns, appName))
}

func (c *cache) GetDeploymentsListByNamespaceAndAppName(ns, appName string) []unstructured.Unstructured {
	return c.Deployment(ns).ByIndex(ApplicationIndex, fmt.Sprintf("%s/%s", ns, appName))
}

func (c *cache) GetConfigMapByNamespaceAndName(ns, name string) (unstructured.Unstructured, error) {
	obj, err := c.ConfigMap(ns).Lister().Get(name)
	if err != nil {
		return unstructured.Unstructured{}, errors.WithStack(err)
	}
	return *obj.(*unstructured.Unstructured).DeepCopy(), nil
}

func (c *cache) GetServiceByNamespaceAndName(ns, name string) (unstructured.Unstructured, error) {
	obj, err := c.Service(ns).Lister().Get(name)
	if err != nil {
		return unstructured.Unstructured{}, errors.WithStack(err)
	}
	return *obj.(*unstructured.Unstructured).DeepCopy(), nil
}

func (c *cache) GetVirtualServiceByNamespaceAndName(ns, name string) (unstructured.Unstructured, error) {
	obj, err := c.VirtualService(ns).Lister().Get(name)
	if err != nil {
		return unstructured.Unstructured{}, errors.WithStack(err)
	}
	return *obj.(*unstructured.Unstructured).DeepCopy(), nil
}

func (c *cache) GetSecretByNamespaceAndName(ns, name string) (unstructured.Unstructured, error) {
	obj, err := c.Secret(ns).Lister().Get(name)
	if err != nil {
		return unstructured.Unstructured{}, errors.WithStack(err)
	}
	return *obj.(*unstructured.Unstructured).DeepCopy(), nil
}

func (c *cache) GetDeploymentByNamespaceAndName(ns, name string) (unstructured.Unstructured, error) {
	obj, err := c.Deployment(ns).Lister().Get(name)
	if err != nil {
		return unstructured.Unstructured{}, errors.WithStack(err)
	}
	return *obj.(*unstructured.Unstructured).DeepCopy(), nil
}

func (c *cache) GetAppConfigByNamespace(ns string) []unstructured.Unstructured {
	return c.Secret(ns).ByIndex(ApplicationConfigIndex, ns)
}

func (c *cache) GetNamespaceListBySelector(selector labels.Selector) []unstructured.Unstructured {
	objs, err := c.Namespace().Lister().List(selector)
	ret := make([]unstructured.Unstructured, len(objs))
	if err == nil {
		for i := range objs {
			ret[i] = *objs[i].(*unstructured.Unstructured).DeepCopy()
		}
	}
	return ret
}

func (c *cache) GetListByKindAndNamespace(kind, ns string) []unstructured.Unstructured {
	switch kind {
	case Deployment:
		return c.Deployment(ns).GetList()
	case Secret:
		return c.Secret(ns).GetList()
	case ConfigMap:
		return c.ConfigMap(ns).GetList()
	case Service:
		return c.Service(ns).GetList()
	case VirtualService:
		return c.VirtualService(ns).GetList()
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

	smap := make(map[string]string)
	services := c.MatchServicesByWorkload(r)
	for _, s := range services {
		smap[s.GetName()] = s.GetName()
		smap[fmt.Sprintf("%s.%s.%s.%s",
			s.GetName(), s.GetNamespace(), "svc", DefaultClusterDomain)] = s.GetName()
	}

	vs := c.GetVirtualServicesListByNamespace(ns)
	for _, v := range vs {
		hosts, _, _ := unstructured.NestedStringSlice(v.UnstructuredContent(), "spec", "hosts")
		for _, host := range hosts {
			svc, ok := smap[host]
			if !ok {
				continue
			}
			if _, ok := vmap[svc]; ok {
				vmap[svc] = append(vmap[svc], v)
				continue
			}
			vmap[svc] = []unstructured.Unstructured{v}
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
		if a[_const.NocalhostApplicationName] == appName {
			return true
		}
		if a[_const.HelmReleaseName] == appName {
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
		if a[_const.NocalhostApplicationName] == appName {
			return false
		}
		if a[_const.HelmReleaseName] == appName {
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
		if _, ok := am[a[_const.NocalhostApplicationName]]; ok {
			return true
		}
		if _, ok := am[a[_const.HelmReleaseName]]; ok {
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
		if _, ok := am[a[_const.NocalhostApplicationName]]; ok {
			return false
		}
		if _, ok := am[a[_const.HelmReleaseName]]; ok {
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
