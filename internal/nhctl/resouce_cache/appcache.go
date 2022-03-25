/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package resouce_cache

import (
	"crypto/sha1"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/pkg/nhctl/k8sutils"
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

func (r *ResourceEventHandlerFuncs) handleAddOrUpdate() {
	// namespace --> appName
	var namespaceToApp = sync.Map{} // key: namespace value: appName
	list := r.informer.Informer().GetStore().List()
	// sort by namespace
	for _, i := range list {
		object, ok := i.(metav1.Object)
		if ok && len(object.GetNamespace()) != 0 {
			v, _ := namespaceToApp.LoadOrStore(object.GetNamespace(), sets.NewString())
			v.(sets.String).Insert(getAppName(i)) // Add app name
		}
	}

	for _, i := range list {
		object, ok := i.(metav1.Object)
		if ok && len(object.GetNamespace()) != 0 {
			kindToAppMap, _ := maps.LoadOrStore(r.toKey(object.GetNamespace()), &sync.Map{})
			kindApp, _ := kindToAppMap.(*sync.Map).LoadOrStore(r.Gvr.Resource, newAppSet()) // R: AppSet
			set := kindApp.(*appSet)
			if value, loaded := namespaceToApp.LoadAndDelete(object.GetNamespace()); loaded {
				set.lock.Lock()
				set.set, _ = value.(sets.String)
				set.lock.Unlock()
			}
		}
	}
}

func (r *ResourceEventHandlerFuncs) OnAdd(interface{}) {
	r.timeUp(r.handleAddOrUpdate)
}

func (r *ResourceEventHandlerFuncs) OnUpdate(_, _ interface{}) {
	r.timeUp(r.handleAddOrUpdate)
}

func (r *ResourceEventHandlerFuncs) OnDelete(obj interface{}) {
	// uninstall application will disconnect vpn
	go func() {
		// if vpn reverse type is pod, it will delete origin pod, and create a new pod with same name
		objectTemp, ok := obj.(metav1.Object)
		if ok && !("pods" == r.Gvr.Resource && objectTemp.GetOwnerReferences() != nil) {
			name := fmt.Sprintf("%s.%s.%s/%s", r.Gvr.Resource, r.Gvr.Version, r.Gvr.Group, objectTemp.GetName())
			if client, err := daemon_client.GetDaemonClient(false); err == nil {
				path := k8sutils.GetOrGenKubeConfigPath(string(r.kubeconfigBytes))
				_ = client.SendVPNOperateCommand(path, objectTemp.GetNamespace(), command.DisConnect, name, nil)
			}
		}
	}()

	// kubeconfig+namespace --> appName
	var namespaceToApp = sync.Map{}
	for _, i := range r.informer.Informer().GetStore().List() {
		object, ok := i.(metav1.Object)
		if ok && len(object.GetNamespace()) != 0 {
			v, _ := namespaceToApp.LoadOrStore(r.toKey(object.GetNamespace()), sets.NewString())
			v.(sets.String).Insert(getAppName(i))
		}
	}
	maps.Range(func(uniqueKey, resourcesMap interface{}) bool {
		// resource --> appName
		resourcesMap.(*sync.Map).Range(func(resource, apps interface{}) bool {
			if r.Gvr.Resource == resource.(string) {
				v, _ := namespaceToApp.LoadOrStore(uniqueKey, sets.NewString())
				apps.(*appSet).lock.Lock()
				apps.(*appSet).set = sets.NewString(v.(sets.String).List()...)
				apps.(*appSet).lock.Unlock()
			}
			return true
		})
		return true
	})
}
