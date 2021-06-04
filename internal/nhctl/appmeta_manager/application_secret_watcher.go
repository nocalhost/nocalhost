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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	profile2 "nocalhost/internal/nhctl/profile"
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

func (aws *applicationSecretWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if secret, ok := obj.(*v1.Secret); ok {
		return aws.join(secret)
	} else {
		errInfo := fmt.Sprintf(
			"Fetching secret with key %s but "+
				"could not cast to secret: %v", key, obj,
		)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (aws *applicationSecretWatcher) Delete(key string) error {
	appName, err := appmeta.GetApplicationName(key)
	if err != nil {
		return err
	}

	aws.left(appName)
	return nil
}

func (aws *applicationSecretWatcher) WatcherInfo() string {
	return fmt.Sprintf("'Secret - ns:%s'", aws.ns)
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
				Event:           event,
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
	delete(asw.applicationMetas, appName)

	for _, event := range *devMetaBefore.Events(devMetaCurrent) {
		EventPush(
			&ApplicationEventPack{
				Event:           event,
				Ns:              asw.ns,
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
	for _, meta := range asw.applicationMetas {
		result = append(result, meta)
	}
	return
}

// prevent other func change the application meta
// caution!!!!!
func (asw *applicationSecretWatcher) GetApplicationMeta(application, ns string) *appmeta.ApplicationMeta {
	if asw != nil && asw.applicationMetas[application] != nil {
		return asw.applicationMetas[application]
	} else {

		return &appmeta.ApplicationMeta{
			ApplicationState:   appmeta.UNINSTALLED,
			Ns:                 ns,
			Application:        application,
			DepConfigName:      "",
			PreInstallManifest: "",
			Manifest:           "",
			DevMeta:            appmeta.ApplicationDevMeta{},
			Config:             &profile2.NocalHostAppConfigV2{},
		}
	}
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
		context.TODO(),
		metav1.ListOptions{FieldSelector: "type=" + appmeta.SecretType},
	)

	// if err occur while list
	// the err can be ignored
	if err != nil {
		log.ErrorE(err, "")
	} else {
		for _, item := range list.Items {
			if err := asw.join(&item); err != nil {
				return err
			}
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
