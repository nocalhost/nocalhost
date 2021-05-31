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

const (
	byNamespace   = "byNamespace"
	byApplication = "byApplication"
	byAppAndNs    = "byAppAndNs"
)

// cache Searcher for each kubeconfig
var searchMap = NewLRU(10, func(i interface{}) { i.(*Searcher).Stop() })
var lock sync.Mutex

type Searcher struct {
	kubeconfig      string
	informerFactory informers.SharedInformerFactory
	supportSchema   map[string]schema.GroupVersionResource
	mapper          meta.RESTMapper
	// is namespaced resource or cluster resource
	namespaced  map[string]bool
	stopChannel chan struct{}
}

func GetSupportGroupVersionResource(kubeconfigBytes []byte) (
	[]schema.GroupVersionResource, map[string]schema.GroupVersionResource, map[string]bool) {
	config, _ := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	Clients, _ := kubernetes.NewForConfig(config)
	g, v, _ := Clients.ServerGroupsAndResources()

	preferredVersion := make(map[string]*metav1.GroupVersionForDiscovery)
	for _, gg := range g {
		preferredVersion[gg.PreferredVersion.GroupVersion] = &gg.PreferredVersion
	}
	nameToUniqueName := make(map[string]string)
	uniqueNameToGroupVersion := make(map[string]string)
	namespaced := make(map[string]bool)
	for _, version := range v {
		if preferredVersion[version.GroupVersion] != nil {
			for _, resource := range version.APIResources {
				if len(resource.ShortNames) != 0 {
					nameToUniqueName[resource.ShortNames[0]] = resource.Name
					nameToUniqueName[resource.Name] = resource.Name
					nameToUniqueName[resource.Kind] = resource.Name
					nameToUniqueName[strings.ToLower(resource.Kind)] = resource.Name
					uniqueNameToGroupVersion[resource.Name] = version.GroupVersion
					namespaced[resource.Name] = resource.Namespaced
				}
			}
		}
	}

	gvrList := make([]schema.GroupVersionResource, 0, len(uniqueNameToGroupVersion))
	uniqueNameToGVR := make(map[string]schema.GroupVersionResource)
	for resource, groupVersion := range uniqueNameToGroupVersion {
		if parseGroupVersion, err := schema.ParseGroupVersion(groupVersion); err == nil {
			groupVersionResource := parseGroupVersion.WithResource(resource)
			gvrList = append(gvrList, groupVersionResource)
			uniqueNameToGVR[resource] = groupVersionResource
		}
	}

	for name, uniqueName := range nameToUniqueName {
		if !uniqueNameToGVR[uniqueName].Empty() {
			uniqueNameToGVR[name] = uniqueNameToGVR[uniqueName]
		}
	}
	return gvrList, uniqueNameToGVR, namespaced
}

func GetSearcher(kubeconfigBytes string, namespace string, isCluster bool) (*Searcher, error) {
	lock.Lock()
	defer lock.Unlock()
	// calculate kubeconfig content's sha value as unique cluster id
	h := sha1.New()
	h.Write([]byte(kubeconfigBytes))
	sum := string(h.Sum([]byte(namespace)))
	searcher, exist := searchMap.Get(sum)
	if !exist || searcher == nil {
		config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigBytes))
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

		indexers := cache.Indexers{
			byNamespace:   byNamespaceFunc,
			byApplication: byApplicationFunc,
			byAppAndNs:    byNamespaceAndAppFunc}

		gvrList, name2gvr, namespaced := GetSupportGroupVersionResource([]byte(kubeconfigBytes))
		for _, gvr := range gvrList {
			// informer not support those two kinds of resource
			if gvr.Resource == "componentstatuses" || gvr.Resource == "customresourcedefinitions" {
				continue
			}
			informer, err := informerFactory.ForResource(gvr)
			if err != nil {
				log.Warnf("can't create informer for resource: %v, error info: %v, ignored", gvr, err)
				continue
			}
			if err = informer.Informer().AddIndexers(indexers); err != nil {
				log.WarnE(err, "informer add indexers error")
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
			t := time.NewTicker(time.Second * 3)
			<-t.C
			firstSyncChannel <- struct{}{}
		}()
		<-firstSyncChannel

		gr, _ := restmapper.GetAPIGroupResources(Clients)

		newSearcher := &Searcher{
			kubeconfig:      kubeconfigBytes,
			informerFactory: informerFactory,
			supportSchema:   name2gvr,
			mapper:          restmapper.NewDiscoveryRESTMapper(gr),
			namespaced:      namespaced,
			stopChannel:     stopChannel,
		}
		searchMap.Add(sum, newSearcher)
	}
	searcher, _ = searchMap.Get(sum)
	return searcher.(*Searcher), nil
}

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

func byNamespaceFunc(obj interface{}) ([]string, error) {
	return []string{obj.(metav1.Object).GetNamespace()}, nil
}

func byApplicationFunc(obj interface{}) ([]string, error) {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		log.Error(err)
		return []string{}, err
	}
	return []string{getAppName(metadata.GetAnnotations())}, nil
}

func byNamespaceAndAppFunc(obj interface{}) ([]string, error) {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		log.Error(err)
		return []string{nsResource("default", nocalhost.DefaultNocalhostApplication)}, nil
	}
	return []string{nsResource(metadata.GetNamespace(), getAppName(metadata.GetAnnotations()))}, nil
}

func getAppName(annotations map[string]string) string {
	if annotations != nil && annotations[nocalhost.NocalhostApplicationName] != "" {
		return annotations[nocalhost.NocalhostApplicationName]
	}
	if annotations != nil && annotations[nocalhost.HelmReleaseName] != "" {
		return annotations[nocalhost.HelmReleaseName]
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

	namespaced   bool
	resourceName string
	appName      string
	ns           string
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
		if c.appName == "" || c.appName == getAppName(item.(metav1.Object).GetAnnotations()) {
			return append(data, item), nil
		}
		return
	}
	var indexName, indexValue string
	if c.ns != "" && c.appName != "" {
		indexName = byAppAndNs
		indexValue = nsResource(c.ns, c.appName)
	} else if c.appName != "" {
		indexName = byApplication
		indexValue = c.appName
	} else if c.ns != "" {
		indexName = byNamespace
		indexValue = c.ns
	} else {
		indexName = ""
		indexValue = ""
	}
	if indexName != "" && indexValue != "" {
		data, e = info.GetIndexer().ByIndex(indexName, indexValue)
		if e == nil {
			SortByNameAsc(data)
		}
	} else {
		data = info.GetIndexer().List()
		SortByNameAsc(data)
	}
	return
}
