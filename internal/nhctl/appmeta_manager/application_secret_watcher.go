/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package appmeta_manager

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/flowcontrol"

	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

type applicationSecretWatcher struct {
	// todo recreate ASW if kubeConfig changed
	configBytes []byte
	ns          string

	applicationMetas map[string]*appmeta.ApplicationMeta
	lock             sync.Mutex
	quit             chan bool

	watchController *watcher.Controller
}

func (asw *applicationSecretWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if secret, ok := obj.(*v1.Secret); ok {
		return asw.join(secret)
	} else {
		errInfo := fmt.Sprintf(
			"Fetching secret with key %s but "+
				"could not cast to secret: %v", key, obj,
		)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (asw *applicationSecretWatcher) Delete(key string) error {
	appName, err := appmeta.GetApplicationName(key)
	if err != nil {
		return err
	}

	asw.left(appName)
	return nil
}

func (asw *applicationSecretWatcher) WatcherInfo() string {
	return fmt.Sprintf("'Secret - ns:%s'", asw.ns)
}

func (asw *applicationSecretWatcher) join(secret *v1.Secret) error {
	devMetaBefore := appmeta.ApplicationDevMeta{}
	devMetaCurrent := appmeta.ApplicationDevMeta{}

	asw.lock.Lock()
	defer asw.lock.Unlock()

	current, err := appmeta.Decode(secret)
	if err != nil {
		return err
	}
	appName := current.Application

	if before, ok := asw.applicationMetas[appName]; ok && before != nil {
		devMetaBefore = before.GetApplicationDevMeta()
	}

	devMetaCurrent = current.DevMeta
	asw.applicationMetas[appName] = current

	for _, event := range *devMetaBefore.Events(devMetaCurrent) {
		EventPush(
			&ApplicationEventPack{
				Event: event,
				//Nid:             current.NamespaceId,
				Ns:              asw.ns,
				AppName:         appName,
				KubeConfigBytes: asw.configBytes,
			},
		)
	}
	return nil
}

func (asw *applicationSecretWatcher) left(appName string) {
	devMetaBefore := appmeta.ApplicationDevMeta{}
	devMetaCurrent := appmeta.ApplicationDevMeta{}

	asw.lock.Lock()
	defer asw.lock.Unlock()

	if before, ok := asw.applicationMetas[appName]; ok {
		devMetaBefore = before.GetApplicationDevMeta()
	}
	//m := asw.applicationMetas[appName]
	delete(asw.applicationMetas, appName)

	for _, event := range *devMetaBefore.Events(devMetaCurrent) {
		EventPush(
			&ApplicationEventPack{
				Event: event,
				Ns:    asw.ns,
				//Nid:             m.NamespaceId,
				AppName:         appName,
				KubeConfigBytes: asw.configBytes,
			},
		)
	}
}

func NewApplicationSecretWatcher(configBytes []byte, ns string) *applicationSecretWatcher {
	return &applicationSecretWatcher{
		configBytes:      configBytes,
		ns:               ns,
		applicationMetas: map[string]*appmeta.ApplicationMeta{},
		quit:             make(chan bool),
	}
}

func (asw *applicationSecretWatcher) GetApplicationMetas() (result []*appmeta.ApplicationMeta) {
	asw.lock.Lock()
	defer asw.lock.Unlock()

	for _, meta := range asw.applicationMetas {
		result = append(result, meta)
	}
	return
}

// prevent other func change the application meta
// caution!!!!!
func (asw *applicationSecretWatcher) GetApplicationMeta(application string) *appmeta.ApplicationMeta {
	asw.lock.Lock()
	defer asw.lock.Unlock()

	return asw.applicationMetas[application]
}

func (asw *applicationSecretWatcher) Quit() {
	asw.quit <- true
}

func (asw *applicationSecretWatcher) Prepare() error {
	c, err := clientcmd.RESTConfigFromKubeConfig(asw.configBytes)
	if err != nil {
		return err
	}

	// creates the clientset
	c.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(10000, 10000)
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}

	// create the secret watcher
	listWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(), "secrets", asw.ns,
		fields.OneTermEqualSelector("type", appmeta.SecretType),
	)

	controller := watcher.NewController(asw, listWatcher, &v1.Secret{})
	asw.watchController = controller

	// first get all nocalhost secrets for initial
	// ignore error prevent kubeconfig has not permission for get secret
	// ignore fail
	list, err := clientset.CoreV1().Secrets(asw.ns).List(
		context.TODO(), metav1.ListOptions{
			FieldSelector: fields.OneTermEqualSelector("type", appmeta.SecretType).String(),
		},
	)
	if err != nil {
		log.ErrorE(err, "")
		return nil
	}

	for _, v := range list.Items {
		// https://medium.com/swlh/use-pointer-of-for-range-loop-variable-in-go-3d3481f7ffc9
		x := v
		if err = asw.join(&x); err != nil {
			return err
		}
	}

	return nil
}

// todo stop while Ns deleted
// this method will block until error occur
func (asw *applicationSecretWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go asw.watchController.Run(1, stop)
	<-asw.quit
}
