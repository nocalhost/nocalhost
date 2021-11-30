/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"crypto/sha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

// key: ns+kubeconfigBytes --> 1: map[resourceType]appSet
var maps sync.Map

type appSet struct {
	set  sets.String
	lock *sync.RWMutex
}

func toKey(kubeconfigBytes []byte, ns string) string {
	h := sha1.New()
	h.Write(kubeconfigBytes)
	return string(h.Sum([]byte(ns)))
}

func GetAllAppNameByNamespace(kubeconfigBytes []byte, ns string) sets.String {
	result := sets.NewString()
	if load, ok := maps.Load(toKey(kubeconfigBytes, ns)); ok {
		load.(*sync.Map).Range(func(_, value interface{}) bool {
			if value != nil {
				result.Insert(value.(*appSet).allKeys()...)
			}
			return true
		})
	}
	return result
}

func (v *appSet) allKeys() (result []string) {
	v.lock.RLock()
	result = v.set.List()
	v.lock.RUnlock()
	return
}

func newAppSet() *appSet {
	return &appSet{set: sets.NewString(), lock: &sync.RWMutex{}}
}

type ResourceEventHandlerFuncs struct {
	cache.ResourceEventHandler
	informer        informers.GenericInformer
	kubeconfigBytes []byte
	nextTime        time.Time
	Gvr             schema.GroupVersionResource
}

func NewResourceEventHandlerFuncs(resource informers.GenericInformer, kubeconfigBytes []byte, gvr schema.GroupVersionResource) *ResourceEventHandlerFuncs {
	return &ResourceEventHandlerFuncs{
		informer:        resource,
		kubeconfigBytes: kubeconfigBytes,
		nextTime:        time.Now(),
		Gvr:             gvr,
	}
}

func (r *ResourceEventHandlerFuncs) toKey(ns string) string {
	return toKey(r.kubeconfigBytes, ns)
}

func (r *ResourceEventHandlerFuncs) timeUp(f func()) {
	if r.nextTime.Sub(time.Now()).Seconds() > 0 {
		return
	}
	f()
	r.nextTime = time.Now().Add(time.Second * 5)
}

func (r *ResourceEventHandlerFuncs) handle() {
	var m sync.Map
	for _, i := range r.informer.Informer().GetStore().List() {
		object := i.(metav1.Object)
		if len(object.GetNamespace()) != 0 {
			kindToAppMap, _ := maps.LoadOrStore(r.toKey(object.GetNamespace()), &sync.Map{})
			kindApp, _ := kindToAppMap.(*sync.Map).LoadOrStore(r.Gvr.Resource, newAppSet())
			set := kindApp.(*appSet)
			set.lock.Lock()
			if _, loaded := m.LoadOrStore(object.GetNamespace(), true); !loaded {
				set.set = sets.NewString()
			}
			set.set.Insert(getAppName(i))
			set.lock.Unlock()
		}
	}
}

func (r *ResourceEventHandlerFuncs) OnAdd(interface{}) {
	r.timeUp(r.handle)
}

func (r *ResourceEventHandlerFuncs) OnUpdate(_, _ interface{}) {
	r.timeUp(r.handle)
}

func (r *ResourceEventHandlerFuncs) OnDelete(interface{}) {
	r.timeUp(r.handle)
}
