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

package appmeta_manager

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
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
	nsAndKeyWithoutPrefix := strings.Split(key, "sh.helm.release.v1.")

	if len(nsAndKeyWithoutPrefix) == 0 {
		log.Error("Invalid Helm Key while delete event watched, not contain 'sh.helm.release.v1.'.")
		return nil
	}

	var keyWithoutPrefix = nsAndKeyWithoutPrefix[len(nsAndKeyWithoutPrefix)-1]

	elems := strings.Split(keyWithoutPrefix, ".v")

	if len(elems) != 2 {
		log.Error("Invalid Helm Key while delete event watched.")
		return nil
	}

	return hcmw.left(elems[0])
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
		hcmw.clientSet,
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
		hcmw.clientSet,
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

func (hcmw *helmCmWatcher) Prepare() error {
	c, err := clientcmd.RESTConfigFromKubeConfig(hcmw.configBytes)
	if err != nil {
		return err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
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
		return err
	}

	// first get all configmaps for initial
	// and find out the invalid nocalhost application
	// then delete it
	searcher, err := resouce_cache.GetSearcher(hcmw.configBytes, hcmw.ns, false)
	if err != nil {
		log.ErrorE(err, "")
		return nil
	}

	cms, err := searcher.Criteria().
		Namespace(hcmw.ns).
		ResourceType("configmaps").Query()
	if err != nil {
		log.ErrorE(err, "")
		return nil
	}

	helmMap := make(map[string]*v1.ConfigMap)

	for _, configmap := range cms {
		v := configmap.(*v1.ConfigMap)

		// this may cause bug that contains sh.helm.release
		// may not managed by helm
		if strings.Contains(v.Name, "sh.helm.release.v1") {
			helmMap[v.Name] = v
			continue
		}
	}

	return searcher.Criteria().
		Namespace(hcmw.ns).
		ResourceType("secrets").
		Consume(
			func(i []interface{}) error {
				nocalhostMap := make(map[string]*v1.Secret)

				for _, secret := range i {
					v := secret.(*v1.Secret)
					if v.Type == appmeta.SecretType {
						nocalhostMap[v.Name] = v
						continue
					}
				}

				for k, _ := range helmMap {
					delete(nocalhostMap, k)
				}

				// the application need to delete
				for k, _ := range nocalhostMap {
					if app, err := appmeta.GetApplicationName(k);
						err == nil &&
							app != nocalhost.DefaultNocalhostApplication {
						_ = hcmw.clientSet.CoreV1().
							Secrets(hcmw.ns).
							Delete(context.TODO(), k, metav1.DeleteOptions{})
					}
				}

				return nil
			},
		)
}

// todo stop while Ns deleted
// this method will block until error occur
func (hcmw *helmCmWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go hcmw.watchController.Run(1, stop)
	<-hcmw.quit
}
