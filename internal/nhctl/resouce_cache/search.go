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

package resouce_cache

import (
	"crypto/sha1"
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// cache Searcher for each kubeconfig
var searchMap = NewLRU(10, func(i interface{}) { i.(*Searcher).Stop() })
var lock sync.Mutex

type Searcher struct {
	kubeconfig      []byte
	informerFactory informers.SharedInformerFactory
	supportSchema   map[string]schema.GroupVersionResource
	mapper          meta.RESTMapper
	// is namespaced resource or cluster resource
	namespaced  map[string]bool
	stopChannel chan struct{}
}

// GetSupportGroupVersionResource
func GetSupportGroupVersionResource(Clients *kubernetes.Clientset, mapper meta.RESTMapper) (
	[]schema.GroupVersionResource, map[string]schema.GroupVersionResource, map[string]bool) {
	apiResourceLists, _ := Clients.ServerPreferredResources()
	nameToUniqueName := make(map[string]string)

	namespaced := make(map[string]bool)
	gvrList := make([]schema.GroupVersionResource, 0)
	uniqueNameToGVR := make(map[string]schema.GroupVersionResource)

	var resourceNeeded = map[string]string{"namespaces": "namespaces"}
	for _, v := range GroupToTypeMap {
		for _, s := range v.V {
			resourceNeeded[s] = s
		}
	}
	versionResource := schema.GroupVersionResource{}
	for _, resourceList := range apiResourceLists {
		for _, resource := range resourceList.APIResources {
			if uniqueNameToGVR[resource.Name] != versionResource {
				continue
			}
			if resourceNeeded[resource.Name] == "" {
				continue
			}
			nameToUniqueName[resource.Name] = resource.Name
			nameToUniqueName[resource.Kind] = resource.Name
			nameToUniqueName[strings.ToLower(resource.Kind)] = resource.Name
			namespaced[resource.Name] = resource.Namespaced
			if len(resource.ShortNames) != 0 {
				nameToUniqueName[resource.ShortNames[0]] = resource.Name
			}
			if parseGroupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion); err == nil {
				groupVersionResource := parseGroupVersion.WithKind(resource.Kind)
				mapping, _ := mapper.RESTMapping(groupVersionResource.GroupKind(), groupVersionResource.Version)
				gvrList = append(gvrList, mapping.Resource)
				uniqueNameToGVR[resource.Name] = mapping.Resource
			}
		}
	}

	for name, uniqueName := range nameToUniqueName {
		if !uniqueNameToGVR[uniqueName].Empty() {
			uniqueNameToGVR[name] = uniqueNameToGVR[uniqueName]
		}
	}
	return gvrList, uniqueNameToGVR, namespaced
}

// GetSearcher
func GetSearcher(kubeconfigBytes []byte, namespace string, isCluster bool) (*Searcher, error) {
	lock.Lock()
	defer lock.Unlock()
	// calculate kubeconfig content's sha value as unique cluster id
	h := sha1.New()
	h.Write(kubeconfigBytes)
	sum := string(h.Sum([]byte(namespace)))
	searcher, exist := searchMap.Get(sum)
	if !exist || searcher == nil {
		config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return nil, err
		}
		Clients, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}

		var informerFactory informers.SharedInformerFactory
		if !isCluster {
			informerFactory = informers.NewSharedInformerFactoryWithOptions(
				Clients, time.Second*5, informers.WithNamespace(namespace))
		} else {
			informerFactory = informers.NewSharedInformerFactory(Clients, time.Second*5)
		}
		gr, _ := restmapper.GetAPIGroupResources(Clients)
		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		gvrList, name2gvr, namespaced := GetSupportGroupVersionResource(Clients, mapper)
		for _, gvr := range gvrList {
			if _, err = informerFactory.ForResource(gvr); err != nil {
				log.Warnf("can't create informer for resource: %v, error info: %v, ignored",
					gvr.Resource, err.Error())
				continue
			}
		}
		stopChannel := make(chan struct{})
		firstSyncChannel := make(chan struct{})
		informerFactory.Start(stopChannel)
		go func() {
			informerFactory.WaitForCacheSync(firstSyncChannel)
			firstSyncChannel <- struct{}{}
		}()
		go func() {
			t := time.NewTicker(time.Second * 2)
			<-t.C
			firstSyncChannel <- struct{}{}
		}()
		<-firstSyncChannel

		newSearcher := &Searcher{
			kubeconfig:      kubeconfigBytes,
			informerFactory: informerFactory,
			supportSchema:   name2gvr,
			mapper:          mapper,
			namespaced:      namespaced,
			stopChannel:     stopChannel,
		}
		searchMap.Add(sum, newSearcher)
	}
	searcher, _ = searchMap.Get(sum)
	return searcher.(*Searcher), nil
}

// Start
func (s *Searcher) Start() {
	<-s.stopChannel
}

func (s *Searcher) Stop() {
	s.stopChannel <- struct{}{}
}

func (s *Searcher) GetGvr(resourceType string) (schema.GroupVersionResource, error) {
	if !s.supportSchema[resourceType].Empty() {
		return s.supportSchema[resourceType], nil
	}
	if !s.supportSchema[strings.ToLower(resourceType)].Empty() {
		return s.supportSchema[strings.ToLower(resourceType)], nil
	}
	return schema.GroupVersionResource{}, errors.New("Not support resource type: " + resourceType)
}

// e's annotation appName must in appNameRange, other wise app name is not available
func getAppName(e interface{}, availableAppName []string) string {
	annotations := e.(metav1.Object).GetAnnotations()
	var appName string
	if annotations != nil && annotations[nocalhost.NocalhostApplicationName] != "" {
		appName = annotations[nocalhost.NocalhostApplicationName]
	}
	if annotations != nil && annotations[nocalhost.HelmReleaseName] != "" {
		appName = annotations[nocalhost.HelmReleaseName]
	}
	availableAppNameMap := make(map[string]string)
	for _, app := range availableAppName {
		availableAppNameMap[app] = app
	}
	if availableAppNameMap[appName] != "" {
		return appName
	}
	return nocalhost.DefaultNocalhostApplication
}

// vendor/k8s.io/client-go/tools/cache/store.go:99, the reason why using ns/resource to get resource
func nsResource(ns, resourceName string) string {
	return fmt.Sprintf("%s/%s", ns, resourceName)
}

func SortByNameAsc(item []interface{}) {
	sort.SliceStable(item, func(i, j int) bool {
		return item[i].(metav1.Object).GetName() < item[j].(metav1.Object).GetName()
	})
}

func (s *Searcher) Criteria() *criteria {
	return newCriteria(s)
}

type criteria struct {
	search *Searcher
	// those two just needs one is enough
	kind         runtime.Object
	resourceType string

	namespaced       bool
	resourceName     string
	appName          string
	ns               string
	availableAppName []string
}

func newCriteria(search *Searcher) *criteria {
	return &criteria{search: search}
}
func (c *criteria) Namespace(namespace string) *criteria {
	c.ns = namespace
	return c
}

func (c *criteria) AppName(appName string) *criteria {
	c.appName = appName
	return c
}

func (c *criteria) AppNameNotIn(appNames ...string) *criteria {
	var result []string
	for _, appName := range appNames {
		if appName != nocalhost.DefaultNocalhostApplication {
			result = append(result, appName)
		}
	}
	c.availableAppName = result
	return c
}

func (c *criteria) ResourceType(resourceType string) *criteria {
	c.resourceType = resourceType
	if gvr, err := c.search.GetGvr(c.resourceType); err == nil {
		c.namespaced = c.search.namespaced[gvr.Resource]
	}
	return c
}

func (c *criteria) Kind(object runtime.Object) *criteria {
	c.kind = object
	// how to make it more elegant
	if reflect.TypeOf(object).AssignableTo(reflect.TypeOf(&corev1.Namespace{})) {
		c.namespaced = false
	}
	return c
}

func (c *criteria) ResourceName(resourceName string) *criteria {
	c.resourceName = resourceName
	return c
}

func (c *criteria) QueryOne() (interface{}, error) {
	query, err := c.Query()
	if err != nil {
		return nil, err
	}
	if len(query) == 0 {
		return nil, errors.New("not found")
	}
	return query[0], nil
}

// Get Query
func (c *criteria) Query() (data []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
		if gvr, errs := c.search.GetGvr(c.resourceType); errs == nil {
			if kind, err2 := c.search.mapper.KindFor(gvr); err2 == nil {
				for _, d := range data {
					d.(runtime.Object).GetObjectKind().SetGroupVersionKind(kind)
				}
			}
		}
	}()

	if c.search == nil {
		return nil, errors.New("search should not be null")
	}
	if c.resourceType == "" && c.kind == nil {
		return nil, errors.New("resource type and kind should not be null at the same time")
	}
	var info cache.SharedIndexInformer
	if c.kind != nil {
		info = c.search.informerFactory.InformerFor(c.kind, nil)
	} else {
		gvr, err := c.search.GetGvr(c.resourceType)
		if err != nil {
			return nil, errors.Wrapf(err, "not support resource type: %v", c.resourceType)
		}
		informer, err := c.search.informerFactory.ForResource(gvr)
		if err != nil {
			return nil, errors.Wrapf(err, "get informer failed for resource type: %v", c.resourceType)
		}
		info = informer.Informer()
	}
	if info == nil {
		return nil, errors.New("create informer failed, please check your code")
	}

	if !c.namespaced {
		list := info.GetStore().List()
		SortByNameAsc(list)
		return list, nil
	}

	if c.ns != "" && c.resourceName != "" {
		item, exists, err1 := info.GetIndexer().GetByKey(nsResource(c.ns, c.resourceName))
		if !exists {
			return nil, errors.Errorf("not found for resource : %s-%s in namespace: %s", c.resourceType, c.resourceName, c.ns)
		}
		if err1 != nil {
			return nil, errors.Wrap(err1, "search occur error")
		}

		// this is a filter
		if c.appName == "" || c.appName == getAppName(item, c.availableAppName) {
			return append(data, item), nil
		}
		return
	}
	return newFilter(info.GetIndexer().List()).namespace(c.ns).appName(c.availableAppName, c.appName).sort().toSlice(), nil
}

type filter struct {
	element []interface{}
}

func newFilter(element []interface{}) *filter {
	return &filter{element: element}
}

func (n *filter) namespace(namespace string) *filter {
	if namespace == "" {
		return n
	}
	var result []interface{}
	for _, e := range n.element {
		if e.(metav1.Object).GetNamespace() == namespace {
			result = append(result, e)
		}
	}
	n.element = result[0:]
	return n
}

func (n *filter) appName(availableAppName []string, appName string) *filter {
	if appName == "" {
		return n
	}
	if appName == nocalhost.DefaultNocalhostApplication {
		return n.appNameNotIn(availableAppName)
	}
	var result []interface{}
	for _, e := range n.element {
		if getAppName(e, availableAppName) == appName {
			result = append(result, e)
		}
	}
	n.element = result[0:]
	return n
}

func (n *filter) appNameNotIn(appNames []string) *filter {
	m := make(map[string]string)
	for _, appName := range appNames {
		m[appName] = appName
	}
	var result []interface{}
	for _, e := range n.element {
		appName := getAppName(e, appNames)
		if m[appName] == "" {
			result = append(result, e)
		}
	}
	n.element = result
	return n
}

func (n *filter) sort() *filter {
	sort.SliceStable(n.element, func(i, j int) bool {
		return n.element[i].(metav1.Object).GetName() < n.element[j].(metav1.Object).GetName()
	})
	return n
}

func (n *filter) toSlice() []interface{} {
	return n.element[0:]
}
