/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"context"
	"crypto/sha1"
	"fmt"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

// cache Searcher for each kubeconfig
var searchMap, _ = simplelru.NewLRU(10000, func(_ interface{}, value interface{}) { value.(*Searcher).Stop() })
var lock sync.Mutex
var clusterMap = make(map[string]bool)
var clusterMapLock sync.Mutex

type Searcher struct {
	kubeconfig      []byte
	informerFactory informers.SharedInformerFactory
	// [string]*meta.RESTMapping
	supportSchema *sync.Map
	mapper        meta.RESTMapper
	stopChannel   chan struct{}
	// last used this searcher, for release informer resource
	lastUsedTime time.Time
	timeoutCheck bool
}

// getSupportedSchema return restMapping of each resource, [string]*meta.RESTMapping
func getSupportedSchema(c *kubernetes.Clientset, mapper meta.RESTMapper) (*sync.Map, error) {
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
				if mapping == nil {
					log.Tracef("RESTMapping is nil, gvc is %v", gvk)
					continue
				}
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
	result := &sync.Map{}
	for k, v := range nameToMapping {
		result.Store(k, v)
	}
	return result, nil
}

// GetSearcherWithTimeoutCheck GetSearchWithLRU will cache kubeconfig with LRU
func GetSearcherWithTimeoutCheck(kubeconfigBytes []byte, namespace string) (search *Searcher, err error) {
	return getSearcherWithCache(kubeconfigBytes, namespace, true)
}

func getSearcherWithCache(kubeconfigBytes []byte, namespace string, timeoutCheck bool) (search *Searcher, err error) {
	defer func() {
		if search != nil {
			search.lastUsedTime = time.Now()
		}
	}()
	lock.Lock()
	defer lock.Unlock()
	searcher, exist := searchMap.Get(generateKey(kubeconfigBytes, namespace))
	if !exist || searcher == nil {
		newSearcher, err := initSearcher(kubeconfigBytes, namespace)
		if err != nil {
			return nil, err
		}
		newSearcher.timeoutCheck = timeoutCheck
		searchMap.Add(generateKey(kubeconfigBytes, namespace), newSearcher)
	}
	if searcher, exist = searchMap.Get(generateKey(kubeconfigBytes, namespace)); exist && searcher != nil {
		search = searcher.(*Searcher)
		err = nil
		return
	}
	return nil, errors.New("Error occurs while init informer searcher")
}

// calculate kubeconfig content's sha value as unique cluster id
func generateKey(kubeconfigBytes []byte, namespace string) string {
	h := sha1.New()
	h.Write(kubeconfigBytes)
	// if it's a cluster admin kubeconfig, then generate key without namespace
	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()
	if _, found := clusterMap[string(kubeconfigBytes)]; found {
		return string(h.Sum(nil))
	} else {
		return string(h.Sum([]byte(namespace)))
	}
}

func GetSearcherWithoutTimeoutCheck(kubeconfigBytes []byte, namespace string) (*Searcher, error) {
	searcher, err := getSearcherWithCache(kubeconfigBytes, namespace, false)
	if err == nil && searcher != nil {
		searcher.timeoutCheck = false
	}
	return searcher, nil
}

// initSearcher return a searcher which use informer to cache resource, without cache
func initSearcher(kubeconfigBytes []byte, namespace string) (*Searcher, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, err
	}
	// default value is flowcontrol.NewTokenBucketRateLimiter(5, 10)
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 10000)
	clientset, err1 := kubernetes.NewForConfig(config)
	if err1 != nil {
		return nil, err1
	}

	var informerFactory informers.SharedInformerFactory

	if isClusterAdmin(kubeconfigBytes) {
		informerFactory = informers.NewSharedInformerFactory(clientset, time.Second*5)
		clusterMapLock.Lock()
		clusterMap[string(kubeconfigBytes)] = true
		clusterMapLock.Unlock()
	} else {
		informerFactory = informers.NewSharedInformerFactoryWithOptions(
			clientset, time.Second*5, informers.WithNamespace(namespace),
		)
	}
	gr, err2 := restmapper.GetAPIGroupResources(clientset)
	if err2 != nil {
		return nil, err2
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	restMappingList, err3 := getSupportedSchema(clientset, mapper)
	if err3 != nil {
		return nil, err3
	}

	restMappingList.Range(func(key, value interface{}) bool {
		if value != nil {
			if restMapping, convert := value.(*meta.RESTMapping); convert && restMapping != nil {
				if _, err = informerFactory.ForResource(restMapping.Resource); err != nil {
					if k8serrors.IsForbidden(err) {
						log.Warnf("user account is forbidden to list resource: %v, ignored", restMapping.Resource)
					} else {
						log.Warnf(
							"Can't create informer for resource: %v, error info: %v, ignored",
							restMapping.Resource, err.Error(),
						)
					}
				}
			}
		}
		return true
	})
	//for _, restMapping := range restMappingList {
	//	if _, err = informerFactory.ForResource(restMapping.Resource); err != nil {
	//		if k8serrors.IsForbidden(err) {
	//			log.Warnf("user account is forbidden to list resource: %v, ignored", restMapping.Resource)
	//		} else {
	//			log.Warnf(
	//				"Can't create informer for resource: %v, error info: %v, ignored",
	//				restMapping.Resource, err.Error(),
	//			)
	//		}
	//		continue
	//	}
	//}
	length := 0
	restMappingList.Range(func(key, value interface{}) bool {
		length++
		return true
	})
	stopChannel := make(chan struct{}, length)
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
	return newSearcher, nil
}

// Start wait searcher to close
func (s *Searcher) Start() {
	<-s.stopChannel
}

// Stop to stop the searcher
func (s *Searcher) Stop() {
	length := 0
	s.supportSchema.Range(func(key, value interface{}) bool {
		length++
		return true
	})
	for i := 0; i < length; i++ {
		s.stopChannel <- struct{}{}
	}
}

func (s *Searcher) GetRestMapping(resourceType string) (*meta.RESTMapping, error) {
	if value, found := s.supportSchema.Load(strings.ToLower(resourceType)); found && value != nil {
		if restMapping, convert := value.(*meta.RESTMapping); convert && restMapping != nil {
			return restMapping, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("Can't get restMapping, resource type: %s", resourceType))
}

// e's annotation appName must in appNameRange, otherwise app name is not available
func getAppName(e interface{}, availableAppName []string) string {
	annotations := e.(metav1.Object).GetAnnotations()
	if annotations == nil {
		return _const.DefaultNocalhostApplication
	}
	var appName string
	if len(annotations[_const.NocalhostApplicationName]) != 0 {
		appName = annotations[_const.NocalhostApplicationName]
	}
	if len(annotations[_const.HelmReleaseName]) != 0 {
		appName = annotations[_const.HelmReleaseName]
	}
	for _, app := range availableAppName {
		if app == appName {
			return appName
		}
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

	namespaceScope   bool
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
		c.namespaceScope = mapping.Scope.Name() == meta.RESTScopeNameNamespace
	} else {
		log.Logf("Can not found restMapping for resource type: %s", resourceType)
	}
	return c
}

func (c *criteria) Kind(object runtime.Object) *criteria {
	c.kind = object
	if info, err := c.search.GetRestMapping(reflect.TypeOf(object).Name()); err == nil {
		c.namespaceScope = info.Scope.Name() == meta.RESTScopeNameNamespace
	} else {
		log.Logf("Can not found restMapping for resource: %s", reflect.TypeOf(object).Name())
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
	if len(c.resourceType) == 0 && c.kind == nil {
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

	// resource is clusterScope, not belong to application or namespace
	if !c.namespaceScope {
		list := informer.GetStore().List()
		if len(c.resourceName) != 0 {
			for _, i := range list {
				if i.(metav1.Object).GetName() == c.resourceName {
					return []interface{}{i}, nil
				}
			}
			return []interface{}{}, nil
		}
		SortByNameAsc(list)
		return list, nil
	}

	// if namespace and resourceName is not empty both, using indexer to query data
	if len(c.ns) != 0 && len(c.resourceName) != 0 {
		item, exists, err1 := informer.GetIndexer().GetByKey(nsResource(c.ns, c.resourceName))
		if !exists {
			return nil, errors.Errorf(
				"not found for resource : %s-%s in namespace: %s", c.resourceType, c.resourceName, c.ns,
			)
		}
		if err1 != nil {
			return nil, errors.Wrap(err1, "search occur error")
		}

		// this is a filter, if appName is empty, just return value
		if len(c.appName) == 0 || c.appName == getAppName(item, c.availableAppName) {
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
	if len(namespace) == 0 {
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
	if len(appName) == 0 {
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

func (n *filter) appNameNotIn(appNamesDefaultAppExclude []string) *filter {
	appNameMap := make(map[string]string)
	for _, appName := range appNamesDefaultAppExclude {
		appNameMap[appName] = appName
	}
	var result []interface{}
	for _, e := range n.element {
		appName := getAppName(e, appNamesDefaultAppExclude)
		if appName == _const.DefaultNocalhostApplication || len(appNameMap[appName]) == 0 {
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

// isClusterAdmin judge weather is cluster scope kubeconfig or not
func isClusterAdmin(kubeconfigBytes []byte) bool {
	c, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return false
	}
	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		return false
	}
	arg := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: "*",
				Group:     "*",
				Verb:      "*",
				Name:      "*",
				Version:   "*",
				Resource:  "*",
			},
		},
	}

	response, err := clientSet.AuthorizationV1().SelfSubjectAccessReviews().Create(
		context.TODO(), arg, metav1.CreateOptions{},
	)
	if err != nil {
		return false
	}
	if response == nil {
		return false
	}
	return response.Status.Allowed
}

// RemoveSearcherByKubeconfig remove informer from cache
func RemoveSearcherByKubeconfig(kubeconfigBytes []byte, namespace string) error {
	removeInformer(generateKey(kubeconfigBytes, namespace))
	c, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}
	list, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err == nil && list != nil {
		for _, item := range list.Items {
			removeInformer(generateKey(kubeconfigBytes, item.Namespace))
		}
	}
	return nil
}

func removeInformer(key string) {
	lock.Lock()
	lock.Unlock()
	if searcher, exist := searchMap.Get(key); exist && searcher != nil {
		go func() { searcher.(*Searcher).Stop() }()
		searchMap.Remove(key)
	}
}

// AddSearcherByKubeconfig init informer in advance
func AddSearcherByKubeconfig(kubeconfigBytes []byte, namespace string) error {
	lock.Lock()
	if searcher, exist := searchMap.Get(generateKey(kubeconfigBytes, namespace)); exist && searcher != nil {
		lock.Unlock()
		return nil
	}
	lock.Unlock()
	go func() { _, _ = GetSearcherWithTimeoutCheck(kubeconfigBytes, namespace) }()
	return nil
}

func init() {
	go func() {
		for {
			select {
			case <-time.Tick(time.Minute * 5):
				go func() {
					defer func() {
						lock.Unlock()
						if err := recover(); err != nil {
							log.Warnf("check informer occurs error, err: %v", err)
						}
					}()
					lock.Lock()
					if searchMap != nil && searchMap.Len() > 0 {
						keys := searchMap.Keys()
						for _, key := range keys {
							if get, found := searchMap.Get(key); found && get != nil {
								if s, ok := get.(*Searcher); ok && s != nil {
									t := time.Time{}
									if s.timeoutCheck && s.lastUsedTime != t && time.Now().Sub(s.lastUsedTime).Hours() >= 24 {
										go func() { s.Stop() }()
										searchMap.Remove(key)
									}
								}
							}
						}
						// keep more used 20 informer
						for i := 0; i < searchMap.Len()-20; i++ {
							if key, value, ok := searchMap.GetOldest(); ok && value != nil {
								if s, convert := value.(*Searcher); convert && s != nil {
									if s.timeoutCheck {
										go func() { value.(*Searcher).Stop() }()
										searchMap.Remove(key)
									}
								}
							}
						}
					}
				}()
			}
		}
	}()
}
