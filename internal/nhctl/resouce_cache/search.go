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
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
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

// cache Search for each kubeconfig
var searchMap = NewLRU(10, func(i interface{}) { i.(*Search).Stop() })
var lock sync.Mutex

type Search struct {
	kubeconfig      string
	informerFactory informers.SharedInformerFactory
	supportSchema   map[string]schema.GroupVersionResource
	stopChannel     chan struct{}
}

func GetSupportGroupVersionResource(kubeconfigBytes []byte) ([]schema.GroupVersionResource, map[string]schema.GroupVersionResource) {
	config, _ := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	Clients, _ := kubernetes.NewForConfig(config)
	g, v, _ := Clients.ServerGroupsAndResources()

	preferredVersion := make(map[string]*metav1.GroupVersionForDiscovery)
	for _, gg := range g {
		preferredVersion[gg.PreferredVersion.GroupVersion] = &gg.PreferredVersion
	}
	nameToUniqueName := make(map[string]string)
	uniqueNameToGroupVersion := make(map[string]string)
	for _, version := range v {
		if preferredVersion[version.GroupVersion] != nil {
			for _, resource := range version.APIResources {
				if len(resource.ShortNames) != 0 {
					nameToUniqueName[resource.ShortNames[0]] = resource.Name
					nameToUniqueName[resource.Name] = resource.Name
					nameToUniqueName[resource.Kind] = resource.Name
					nameToUniqueName[strings.ToLower(resource.Kind)] = resource.Name
					uniqueNameToGroupVersion[resource.Name] = version.GroupVersion
				}
			}
		}
	}

	gvrList := make([]schema.GroupVersionResource, 0, len(uniqueNameToGroupVersion))
	uniqueNameToGVR := make(map[string]schema.GroupVersionResource)
	for resource, groupVersion := range uniqueNameToGroupVersion {
		gv := strings.Split(groupVersion, "/")
		var group, version string
		if len(gv) == 0 {
			continue
		} else if len(gv) == 1 {
			version = gv[0]
		} else if len(gv) == 2 {
			group = gv[0]
			version = gv[1]
		}
		gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
		gvrList = append(gvrList, gvr)
		uniqueNameToGVR[resource] = gvr
	}

	for name, uniqueName := range nameToUniqueName {
		if !uniqueNameToGVR[uniqueName].Empty() {
			uniqueNameToGVR[name] = uniqueNameToGVR[uniqueName]
		}
	}
	return gvrList, uniqueNameToGVR
}

func GetSearch(kubeconfigBytes string, namespace string) (*Search, error) {
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

		informerFactory := informers.NewSharedInformerFactoryWithOptions(Clients, time.Second*5, informers.WithNamespace(namespace))
		indexers := cache.Indexers{
			byNamespace:   byNamespaceFunc,
			byApplication: byApplicationFunc,
			byAppAndNs:    byNamespaceAndAppFunc}

		gvrList, name2gvr := GetSupportGroupVersionResource([]byte(kubeconfigBytes))
		for _, gvr := range gvrList {
			// informer not support those two kinds of resource
			if gvr.Resource == "componentstatuses" || gvr.Resource == "customresourcedefinitions" {
				continue
			}
			informer, err := informerFactory.ForResource(gvr)
			checkError(err)
			err = informer.Informer().AddIndexers(indexers)
			checkError(err)
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

		newSearcher := &Search{
			kubeconfig:      kubeconfigBytes,
			informerFactory: informerFactory,
			supportSchema:   name2gvr,
			stopChannel:     stopChannel,
		}
		searchMap.Add(sum, newSearcher)
	}
	searcher, _ = searchMap.Get(sum)
	return searcher.(*Search), nil
}

func (s *Search) Start() {
	<-s.stopChannel
}

func (s *Search) Stop() {
	s.stopChannel <- struct{}{}
}

func (s *Search) GetByApplication(obj runtime.Object, appName string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	return s.informerFactory.InformerFor(obj, nil).GetIndexer().ByIndex(byApplication, appName)
}

func (s *Search) GetByNamespace(obj runtime.Object, namespace string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	return s.informerFactory.InformerFor(obj, nil).GetIndexer().ByIndex(byNamespace, namespace)
}

func (s *Search) GetByAppAndNamespace(obj runtime.Object, app, namespace string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	return s.informerFactory.InformerFor(obj, nil).GetIndexer().ByIndex(byAppAndNs, nsResource(namespace, app))
}

// example, po --> pods, Pods --> pods, pod --> pods, all works
func (s *Search) GetAllByResourceType(resourceType string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}
	return informer.Informer().GetIndexer().List(), nil
}

func (s *Search) GetAllByResourceTypeAndNameAndNs(resourceType, resourceName, ns string) (i interface{}, b bool, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, false, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, false, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}
	return informer.Informer().GetIndexer().GetByKey(nsResource(ns, resourceName))
}

func (s Search) GetAllByResourceTypeAndNs(resourceType, ns string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}
	return informer.Informer().GetIndexer().ByIndex(byNamespace, ns)
}

func (s *Search) GetByResourceAndApplication(resourceType string, appName string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()

	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}

	return informer.Informer().GetIndexer().ByIndex(byApplication, appName)
}

func (s *Search) GetGvr(resourceType string) (schema.GroupVersionResource, error) {
	if !s.supportSchema[resourceType].Empty() {
		return s.supportSchema[resourceType], nil
	}
	if !s.supportSchema[strings.ToLower(resourceType)].Empty() {
		return s.supportSchema[strings.ToLower(resourceType)], nil
	}
	return schema.GroupVersionResource{}, errors.New("Not support resource type: " + resourceType)
}

func (s *Search) GetByResourceAndNamespace(resourceType, resourceName, namespace string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}
	if resourceName == "" {
		return informer.Informer().GetIndexer().ByIndex("byNamespace", namespace)
	} else {
		i, found, err2 := informer.Informer().GetIndexer().GetByKey(nsResource(namespace, resourceName))
		if err2 != nil {
			return nil, err2
		} else if !found {
			return nil, errors.New("not found resource")
		} else {
			return []interface{}{i}, nil
		}
	}
}

func (s *Search) GetByResourceAndAppAndNamespace(resourceType, appName, namespace string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}
	return informer.Informer().GetIndexer().ByIndex(byAppAndNs, nsResource(namespace, appName))
}

func (s *Search) GetByResourceAndNameAndAppAndNamespace(resourceType, resourceName, appName, namespace string) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	gvr, err := s.GetGvr(resourceType)
	if err != nil {
		return nil, errors.New("Not support resource type: " + resourceType)
	}
	informer, err := s.informerFactory.ForResource(gvr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Get informer for resource: %v error", s.supportSchema[resourceType]))
	}
	item, err := informer.Informer().GetIndexer().ByIndex(byAppAndNs, nsResource(namespace, appName))
	if err != nil {
		return nil, err
	}
	if resourceName != "" {
		for _, i := range item {
			if i.(metav1.Object).GetName() == resourceName {
				return []interface{}{i}, nil
			}
		}
		return nil, errors.New("not found")
	}
	return item, nil
}

func (s *Search) GetAllByType(obj runtime.Object) (i []interface{}, e error) {
	defer func() {
		if err := recover(); err != nil {
			e = err.(error)
		}
	}()
	return s.informerFactory.InformerFor(obj, nil).GetIndexer().List(), nil
}

func (s *Search) GetByName(obj runtime.Object, namespace, name string) (item interface{}, exists bool, e error) {
	defer func() {
		if err := recover(); err != nil {
			exists = false
			e = err.(error)
		}
	}()
	return s.informerFactory.InformerFor(obj, nil).GetIndexer().GetByKey(nsResource(namespace, name))
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
	anno := metadata.GetAnnotations()
	if anno == nil || len(anno) == 0 ||
		(anno[nocalhost.NocalhostApplicationName] == "" && anno[nocalhost.HelmReleaseName] == "") {
		return []string{nocalhost.DefaultNocalhostApplication}, nil
	}
	if anno[nocalhost.NocalhostApplicationName] != "" {
		return []string{anno[nocalhost.NocalhostApplicationName]}, nil
	} else {
		return []string{anno[nocalhost.HelmReleaseName]}, nil
	}
}

func byNamespaceAndAppFunc(obj interface{}) ([]string, error) {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		log.Error(err)
		return []string{nsResource("default", nocalhost.DefaultNocalhostApplication)}, nil
	}
	ns := metadata.GetNamespace()
	anno := metadata.GetAnnotations()
	if anno == nil || len(anno) == 0 ||
		(anno[nocalhost.NocalhostApplicationName] == "" && anno[nocalhost.HelmReleaseName] == "") {
		return []string{nsResource(ns, nocalhost.DefaultNocalhostApplication)}, nil
	}
	if anno[nocalhost.NocalhostApplicationName] != "" {
		return []string{nsResource(ns, anno[nocalhost.NocalhostApplicationName])}, nil
	} else {
		return []string{nsResource(ns, anno[nocalhost.HelmReleaseName])}, nil
	}
}

// vendor/k8s.io/client-go/tools/cache/store.go:99, the reason why using ns/resource to get resource
func nsResource(ns, resourceName string) string {
	return fmt.Sprintf("%s/%s", ns, resourceName)
}

func SortByCreateTimestampAsc(item []interface{}) []interface{} {
	sort.SliceStable(item, func(i, j int) bool {
		t1 := item[i].(metav1.Object).GetCreationTimestamp().UnixNano()
		t2 := item[j].(metav1.Object).GetCreationTimestamp().UnixNano()
		if t1 > t2 {
			return false
		}
		return true
	})
	return item
}

func checkError(err error) {
	if err != nil {
		// ignore
	}
}
