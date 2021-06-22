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

	return hws.left(elems[0])
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
		hws.clientSet,
	); err != nil {
		log.TLogf(
			"Watcher", "Helm application found from secret: %s,"+
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
		hws.clientSet,
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

func (hws *helmSecretWatcher) Prepare() error {
	c, err := clientcmd.RESTConfigFromKubeConfig(hws.configBytes)
	if err != nil {
		return err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
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
	hws.clientSet, err = kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}

	// first get all secrets for initial
	// and find out the invalid nocalhost application
	// then delete it
	searcher, err := resouce_cache.GetSearcher(hws.configBytes, hws.ns, false)
	if err != nil {
		log.ErrorE(err, "")
		return nil
	}

	return searcher.Criteria().
		Namespace(hws.ns).
		ResourceType("secrets").
		Consume(
			func(i []interface{}) error {
				nocalhostMap := make(map[string]*v1.Secret)
				helmMap := make(map[string]*v1.Secret)

				for _, secret := range i {
					v := secret.(*v1.Secret)
					if v.Type == appmeta.SecretType {
						nocalhostMap[v.Name] = v
						continue
					}

					// this may cause bug that contains sh.helm.release
					// may not managed by helm
					if strings.Contains(v.Name, "sh.helm.release.v1") {
						helmMap[v.Name] = v
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
						_ = hws.clientSet.CoreV1().
							Secrets(hws.ns).
							Delete(context.TODO(), k, metav1.DeleteOptions{})
					}
				}

				return nil
			},
		)
}

// todo stop while Ns deleted
// this method will block until error occur
func (hws *helmSecretWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go hws.watchController.Run(1, stop)
	<-hws.quit
}
