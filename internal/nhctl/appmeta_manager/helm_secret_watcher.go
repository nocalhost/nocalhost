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
	"k8s.io/client-go/util/flowcontrol"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

type helmSecretWatcher struct {
	// todo recreate HSW if kubeConfig changed
	configBytes []byte
	ns          string

	lock sync.Mutex
	quit chan bool

	watchController *watcher.Controller
	clientSet       *kubernetes.Clientset
}

func (hws *helmSecretWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if secret, ok := obj.(*v1.Secret); ok {
		return hws.join(secret)
	} else {
		errInfo := fmt.Sprintf(
			"Fetching secret with key %s but "+
				"could not cast to secret: %v", key, obj,
		)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (hws *helmSecretWatcher) Delete(key string) error {
	rlsName, err := GetRlsNameFromKey(key)
	if err != nil {
		log.Error(err)
		return nil
	}

	return hws.left(rlsName)
}

func (hws *helmSecretWatcher) WatcherInfo() string {
	return fmt.Sprintf("'Helm-Secret - ns:%s'", hws.ns)
}

func (hws *helmSecretWatcher) join(secret *v1.Secret) error {
	hws.lock.Lock()
	defer hws.lock.Unlock()

	// try to new application from helm configmap
	if err := tryNewAppFromHelmRelease(
		string(secret.Data["release"]),
		hws.ns,
		hws.configBytes,
	); err != nil {
		log.Debugf(
			"Helm application found from secret: %s,"+
				" but error occur while processing: %s", secret.Name, err,
		)
	}
	return nil
}

func (hws *helmSecretWatcher) left(appName string) error {
	hws.lock.Lock()
	defer hws.lock.Unlock()

	// try to new application from helm configmap
	if err := tryDelAppFromHelmRelease(
		appName,
		hws.ns,
		hws.configBytes,
	); err != nil {
		log.TLogf(
			"Watcher", "Helm application '%s' is deleted,"+
				" but error occur while processing: %s", appName, err,
		)
	}
	return nil
}

func NewHelmSecretWatcher(configBytes []byte, ns string) *helmSecretWatcher {
	return &helmSecretWatcher{
		configBytes: configBytes,
		ns:          ns,
		quit:        make(chan bool),
	}
}

func (hws *helmSecretWatcher) Quit() {
	hws.quit <- true
}

func (hws *helmSecretWatcher) Prepare() (existRelease []string, err error) {
	c, err := clientcmd.RESTConfigFromKubeConfig(hws.configBytes)
	if err != nil {
		return
	}

	// creates the clientset
	c.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 10000)
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return
	}

	// create the secret watcher
	listWatcher := cache.NewFilteredListWatchFromClient(
		clientset.CoreV1().RESTClient(), "secrets", hws.ns,
		func(options *metav1.ListOptions) {
			options.LabelSelector = kblabels.Set{"owner": "helm"}.AsSelector().String()
		},
	)

	controller := watcher.NewController(hws, listWatcher, &v1.Secret{})
	hws.watchController = controller

	// creates the clientset
	hws.clientSet = clientset

	//// first get all secrets for initial
	//// and find out the invalid nocalhost application
	//ss, err := clientset.CoreV1().Secrets(hws.ns).List(context.TODO(), metav1.ListOptions{})
	//if err != nil {
	//	log.ErrorE(err, "")
	//	return
	//}
	//
	//for _, v := range ss.Items {
	//	// this may cause bug that contains sh.helm.release
	//	// may not managed by helm
	//	if strings.Contains(v.Name, "sh.helm.release.v1") {
	//		if release, err := DecodeRelease(string(v.Data["release"])); err == nil && release.Info.Deleted == "" {
	//			if rlsName, err := GetRlsNameFromKey(v.Name); err == nil {
	//				existRelease = append(existRelease, rlsName)
	//			}
	//		}
	//	}
	//}

	return
}

// todo stop while Ns deleted
// this method will block until error occur
func (hws *helmSecretWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go hws.watchController.Run(1, stop)
	<-hws.quit
}
