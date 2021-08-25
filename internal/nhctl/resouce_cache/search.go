/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"crypto/sha1"
	"fmt"
	"github.com/hashicorp/golang-lru/simplelru"
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
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
	"reflect"
	"sort"
	"strings"
	"time"
)

// cache Searcher for each kubeconfig
var searchMap, _ = simplelru.NewLRU(15, func(_ interface{}, value interface{}) { value.(*Searcher).Stop() })

type Searcher struct {
	kubeconfig      []byte
	informerFactory informers.SharedInformerFactory
	supportSchema   map[string]*meta.RESTMapping
	mapper          meta.RESTMapper
	stopChannel     chan struct{}
}

// GetSupportedSchema
func GetSupportedSchema(c *kubernetes.Clientset, mapper meta.RESTMapper) (map[string]*meta.RESTMapping, error) {
	var resourceNeeded = map[string]string{"namespaces": "namespaces"}
	for _, v := range GroupToTypeMap {
		for _, s := range v.V {
			resourceNeeded[s] = s
		}
	}
	apiResourceLists, err := c.ServerPreferredResources()
	if err != nil && len(apiResourceLists) == 0 {
		return nil, err
	}
	nameToMapping := make(map[string]*meta.RESTMapping)
	for _, resourceList := range apiResourceLists {
		for _, resource := range resourceList.APIResources {
			if resourceNeeded[resource.Name] == "" {
				continue
			}
			if nameToMapping[resource.Name] != nil {
				log.Logf("Already exist resource type: %s, restMapping: %v",
					resource.Name, nameToMapping[resource.Name])
				continue
			}
			if groupVersion, err := schema.ParseGroupVersion(resourceList.GroupVersion); err == nil {
				gvk := groupVersion.WithKind(resource.Kind)
				mapping, _ := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
				nameToMapping[resource.Name] = mapping
				nameToMapping[resource.Kind] = mapping
				nameToMapping[strings.ToLower(resource.Kind)] = mapping
				for _, name := range resource.ShortNames {
					nameToMapping[name] = mapping
				}
			}
		}
	}
	if len(nameToMapping) == 0 {
		return nil, errors.New("RestMapping is empty, this should not happened")
	}
	return nameToMapping, nil
}

// GetSearcher
func GetSearcher(kubeconfigBytes []byte, namespace string, isCluster bool) (*Searcher, error) {
	// calculate kubeconfig content's sha value as unique cluster id
	h := sha1.New()
	h.Write(kubeconfigBytes)
	key := string(h.Sum([]byte(namespace)))
	searcher, exist := searchMap.Get(key)
	if !exist || searcher == nil {
		config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
		if err != nil {
			return nil, err
		}
		clientset, err1 := kubernetes.NewForConfig(config)
		if err1 != nil {
			return nil, err1
		}

		var informerFactory informers.SharedInformerFactory
		if !isCluster {
			informerFactory = informers.NewSharedInformerFactoryWithOptions(
				clientset, time.Second*5, informers.WithNamespace(namespace),
			)
		} else {
			informerFactory = informers.NewSharedInformerFactory(clientset, time.Second*5)
		}
		gr, err2 := restmapper.GetAPIGroupResources(clientset)
		if err2 != nil {
			return nil, err2
		}
		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		restMappingList, err3 := GetSupportedSchema(clientset, mapper)
		if err3 != nil {
			return nil, err3
		}
		for _, restMapping := range restMappingList {
			if _, err = informerFactory.ForResource(restMapping.Resource); err != nil {
				log.Warnf(
					"Can't create informer for resource: %v, error info: %v, ignored",
					restMapping.Resource, err.Error(),
				)
				continue
			}
		}
		stopChannel := make(chan struct{}, len(restMappingList))
		firstSyncChannel := make(chan struct{}, 2)
		informerFactory.Start(stopChannel)
		go func() {
			informerFactory.WaitForCacheSync(firstSyncChannel)
			firstSyncChannel <- struct{}{}
		}()
		go func() {
			t := time.NewTicker(time.Second * 3)
			<-t.C
			firstSyncChannel <- struct{}{}
		}()
		<-firstSyncChannel

		newSearcher := &Searcher{
			kubeconfig:      kubeconfigBytes,
			informerFactory: informerFactory,
			supportSchema:   restMappingList,
			mapper:          mapper,
			stopChannel:     stopChannel,
		}
		if searcher, exist = searchMap.Get(key); !exist || searcher == nil {
			searchMap.Add(key, newSearcher)
		}
	}
	if searcher, exist = searchMap.Get(key); exist && searcher != nil {
		return searcher.(*Searcher), nil
	}
	return nil, errors.New("Error occurs while init informer searcher")
}

// Start
func (s *Searcher) Start() {
	<-s.stopChannel
}

func (s *Searcher) Stop() {
	for i := 0; i < len(s.supportSchema); i++ {
		s.stopChannel <- struct{}{}
	}
}

func (s *Searcher) GetRestMapping(resourceType string) (*meta.RESTMapping, error) {
	if s.supportSchema[strings.ToLower(resourceType)] != nil {
		return s.supportSchema[strings.ToLower(resourceType)], nil
	}
	return nil, errors.New(fmt.Sprintf("Can't get restMapping, resource type: %s", resourceType))
}

// e's annotation appName must in appNameRange, other wise app name is not available
func getAppName(e interface{}, availableAppName []string) string {
	annotations := e.(metav1.Object).GetAnnotations()
	var appName string
	if annotations != nil && annotations[_const.NocalhostApplicationName] != "" {
		appName = annotations[_const.NocalhostApplicationName]
	}
	if annotations != nil && annotations[_const.HelmReleaseName] != "" {
		appName = annotations[_const.HelmReleaseName]
	}
	availableAppNameMap := make(map[string]string)
	for _, app := range availableAppName {
		availableAppNameMap[app] = app
	}
	if availableAppNameMap[appName] != "" {
		return appName
	}
	return _const.DefaultNocalhostApplication
}

// vendor/k8s.io/client-go/tools/cache/store.go:99, the reason why using ns/resource to get resource
func nsResource(ns, resourceName string) string {
	return fmt.Sprintf("%s/%s", ns, resourceName)
}

func SortByNameAsc(item []interface{}) {
	sort.SliceStable(
		item, func(i, j int) bool {
			return item[i].(metav1.Object).GetName() < item[j].(metav1.Object).GetName()
		},
	)
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
	label            map[string]string
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
		if appName != _const.DefaultNocalhostApplication {
			result = append(result, appName)
		}
	}
	c.availableAppName = result
	return c
}

func (c *criteria) ResourceType(resourceType string) *criteria {
	if mapping, err := c.search.GetRestMapping(resourceType); err == nil {
		c.resourceType = resourceType
		c.namespaced = mapping.Scope.Name() == meta.RESTScopeNameNamespace
	} else {
		log.Logf("Can not found restMapping for resource type: %s", resourceType)
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

func (c *criteria) Label(label map[string]string) *criteria {
	c.label = label
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

func (c *criteria) Consume(consumer func([]interface{}) error) error {
	query, err := c.Query()
	if err != nil {
		return err
	}

	return consumer(query)
}

// Get Query
func (c *criteria) Query() (data []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
		if mapping, errs := c.search.GetRestMapping(c.resourceType); errs == nil {
			for _, d := range data {
				d.(runtime.Object).GetObjectKind().SetGroupVersionKind(mapping.GroupVersionKind)
			}
		}
	}()

	if c.search == nil {
		return nil, errors.New("search should not be null")
	}
	if c.resourceType == "" && c.kind == nil {
		return nil, errors.New("resource type and kind should not be null at the same time")
	}
	var informer cache.SharedIndexInformer
	if c.kind != nil {
		informer = c.search.informerFactory.InformerFor(c.kind, nil)
	} else {
		mapping, err := c.search.GetRestMapping(c.resourceType)
		if err != nil {
			return nil, errors.Wrapf(err, "not support resource type: %v", c.resourceType)
		}
		genericInformer, err := c.search.informerFactory.ForResource(mapping.Resource)
		if err != nil {
			return nil, errors.Wrapf(err, "get informer failed for resource type: %v", c.resourceType)
		}
		informer = genericInformer.Informer()
	}
	if informer == nil {
		return nil, errors.New("create informer failed, please check your code")
	}

	if !c.namespaced {
		list := informer.GetStore().List()
		SortByNameAsc(list)
		return list, nil
	}

	if c.ns != "" && c.resourceName != "" {
		item, exists, err1 := informer.GetIndexer().GetByKey(nsResource(c.ns, c.resourceName))
		if !exists {
			return nil, errors.Errorf(
				"not found for resource : %s-%s in namespace: %s", c.resourceType, c.resourceName, c.ns,
			)
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
	return newFilter(informer.GetIndexer().List()).
		namespace(c.ns).
		appName(c.availableAppName, c.appName).
		label(c.label).
		notLabel(map[string]string{_const.DevWorkloadIgnored: "true"}).
		sort().
		toSlice(), nil
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
	if appName == _const.DefaultNocalhostApplication {
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

// support equals, like: a == b
func (n *filter) label(label map[string]string) *filter {
	n.element = labelSelector(n.element, label, func(v1, v2 string) bool { return v1 == v2 })
	return n
}

// support not equal, like a != b
func (n *filter) notLabel(label map[string]string) *filter {
	n.element = labelSelector(n.element, label, func(v1, v2 string) bool { return v1 != v2 })
	return n
}

func labelSelector(element []interface{}, label map[string]string, f func(string, string) bool) []interface{} {
	var result []interface{}
	for _, e := range element {
		labels := e.(metav1.Object).GetLabels()
		match := true
		for k, v := range label {
			if !f(labels[k], v) {
				match = false
				break
			}
		}
		if match {
			result = append(result, e)
		}
	}
	return result[0:]
}

func (n *filter) sort() *filter {
	sort.SliceStable(
		n.element, func(i, j int) bool {
			return n.element[i].(metav1.Object).GetName() < n.element[j].(metav1.Object).GetName()
		},
	)
	return n
}

func (n *filter) toSlice() []interface{} {
	return n.element[0:]
}
