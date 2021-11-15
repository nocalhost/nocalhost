/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"crypto/sha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"sync"
	"time"
)

var maps sync.Map

type value struct {
	sets.String
	lock *sync.RWMutex
}

func toKey(kubeconfigBytes []byte, ns string) string {
	h := sha1.New()
	h.Write(kubeconfigBytes)
	return string(h.Sum([]byte(ns)))
}

func GetAllAppNameByNamespace(kubeconfigBytes []byte, ns string) []string {
	load, _ := maps.Load(toKey(kubeconfigBytes, ns))
	if load != nil {
		return load.(*value).allKeys()
	}
	return []string{}
}

func (v *value) allKeys() (result []string) {
	v.lock.RLock()
	result = v.List()
	v.lock.RUnlock()
	return
}

func newValue() *value {
	return &value{String: sets.NewString(), lock: &sync.RWMutex{}}
}

type ResourceEventHandlerFuncs struct {
	cache.ResourceEventHandler
	informer        informers.GenericInformer
	kubeconfigBytes []byte
	nextTime        time.Time
}

func NewResourceEventHandlerFuncs(resource informers.GenericInformer, kubeconfigBytes []byte) *ResourceEventHandlerFuncs {
	return &ResourceEventHandlerFuncs{
		informer:        resource,
		kubeconfigBytes: kubeconfigBytes,
		nextTime:        time.Now(),
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
	for _, i := range r.informer.Informer().GetStore().List() {
		object := i.(metav1.Object)
		store, _ := maps.LoadOrStore(r.toKey(object.GetNamespace()), newValue())
		store.(*value).lock.Lock()
		store.(*value).Insert(getAppName(i))
		store.(*value).lock.Unlock()
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
