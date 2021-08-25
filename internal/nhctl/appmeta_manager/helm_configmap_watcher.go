/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package appmeta_manager

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"sync"
)

type helmCmWatcher struct {
	// todo recreate HSW if kubeConfig changed
	configBytes []byte
	ns          string

	lock sync.Mutex
	quit chan bool

	watchController *watcher.Controller
	clientSet       *kubernetes.Clientset
}

func (hcmw *helmCmWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if configMap, ok := obj.(*v1.ConfigMap); ok {
		return hcmw.join(configMap)
	} else {
		errInfo := fmt.Sprintf(
			"Fetching cm with key %s but "+
				"could not cast to configmap: %v", key, obj,
		)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (hcmw *helmCmWatcher) Delete(key string) error {
	rlsName, err := GetRlsNameFromKey(key)
	if err != nil {
		log.Error(err)
		return nil
	}

	return hcmw.left(rlsName)
}

func (hcmw *helmCmWatcher) WatcherInfo() string {
	return fmt.Sprintf("'Helm-Cm - ns:%s'", hcmw.ns)
}

func (hcmw *helmCmWatcher) join(configMap *v1.ConfigMap) error {
	hcmw.lock.Lock()
	defer hcmw.lock.Unlock()

	// try to new application from helm configmap
	if err := tryNewAppFromHelmRelease(
		configMap.Data["release"],
		hcmw.ns,
		hcmw.configBytes,
	); err != nil {
		log.TLogf(
			"Watcher", "Helm application found from cm: %s,"+
				" but error occur while processing: %s", configMap.Name, err,
		)
	}
	return nil
}

func (hcmw *helmCmWatcher) left(appName string) error {
	hcmw.lock.Lock()
	defer hcmw.lock.Unlock()

	// try to new application from helm configmap
	if err := tryDelAppFromHelmRelease(
		appName,
		hcmw.ns,
		hcmw.configBytes,
	); err != nil {
		log.TLogf(
			"Watcher", "Helm application '%s' is deleted,"+
				" but error occur while processing: %s", appName, err,
		)
	}
	return nil
}

func NewHelmCmWatcher(configBytes []byte, ns string) *helmCmWatcher {
	return &helmCmWatcher{
		configBytes: configBytes,
		ns:          ns,
		quit:        make(chan bool),
	}
}

func (hcmw *helmCmWatcher) Quit() {
	hcmw.quit <- true
}

// Prepare for watcher and return the exist helm Release application
func (hcmw *helmCmWatcher) Prepare() (existRelease []string, err error) {

	c, err := clientcmd.RESTConfigFromKubeConfig(hcmw.configBytes)
	if err != nil {
		return
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return
	}

	// create the configmap watcher
	listWatcher := cache.NewFilteredListWatchFromClient(
		clientset.CoreV1().RESTClient(), "configmaps", hcmw.ns,
		func(options *metav1.ListOptions) {
			options.LabelSelector = kblabels.Set{"owner": "helm"}.AsSelector().String()
		},
	)

	controller := watcher.NewController(hcmw, listWatcher, &v1.ConfigMap{})
	hcmw.watchController = controller

	// creates the clientset
	hcmw.clientSet, err = kubernetes.NewForConfig(c)
	if err != nil {
		return
	}

	// first get all configmaps for initial
	// and find out the invalid nocalhost application
	// then delete it
	searcher, err := resouce_cache.GetSearcher(hcmw.configBytes, hcmw.ns, false)
	if err != nil {
		log.ErrorE(err, "")
		return
	}

	cms, err := searcher.Criteria().
		Namespace(hcmw.ns).
		ResourceType("configmaps").Query()
	if err != nil {
		log.ErrorE(err, "")
		return
	}

	for _, configmap := range cms {
		v := configmap.(*v1.ConfigMap)

		// this may cause bug that contains sh.helm.release
		// may not managed by helm
		if strings.Contains(v.Name, "sh.helm.release.v1") {
			if release, err := DecodeRelease(v.Data["release"]); err == nil && release.Info.Deleted == "" {
				if rlsName, err := GetRlsNameFromKey(v.Name); err == nil {
					existRelease = append(existRelease, rlsName)
				}
			}
		}
	}

	return
}

// todo stop while Ns deleted
// this method will block until error occur
func (hcmw *helmCmWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go hcmw.watchController.Run(1, stop)
	<-hcmw.quit
}
