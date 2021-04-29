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

package service_account

import (
	"encoding/json"
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"sync"
)

type roleBindingWatcher struct {
	// todo recreate RBW if kubeConfig changed
	kubeConfig string

	ownNs map[string] /* ns */ *set /* serviceAccount */
	lock  sync.Mutex
	quit  chan bool

	watchController *watcher.Controller
}

func (rbw *roleBindingWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if secret, ok := obj.(*rbacv1.RoleBinding); ok {
		return rbw.join(secret)
	} else {
		errInfo := fmt.Sprintf("Fetching secret with key %s but could not cast to secret: %v", key, obj)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (rbw *roleBindingWatcher) Delete(key string) error {
	appName, err := appmeta.GetApplicationName(key)
	if err != nil {
		return err
	}

	rbw.left(appName)
	return nil
}

func (rbw *roleBindingWatcher) WatcherInfo() string {
	return fmt.Sprintf("'RoleBinding - ns:%s'", rbw.ownNs)
}

func (rbw *roleBindingWatcher) join(rb *rbacv1.RoleBinding) error {
	rbw.lock.Lock()
	defer rbw.lock.Unlock()

	serviceAccounts := newSet()

	for _, subject := range rb.Subjects {
		serviceAccounts.put(subject.Name)
	}
	rbw.ownNs[rb.Namespace] = serviceAccounts
	return nil
}

func (rbw *roleBindingWatcher) left(rbName string) {
	rbw.lock.Lock()
	defer rbw.lock.Unlock()

	log.Info("left:" + rbName)
}

func NewRoleBindingWatcher(kubeConfig string) *roleBindingWatcher {
	return &roleBindingWatcher{
		kubeConfig: kubeConfig,
		ownNs:      map[string] /* serviceAccount */ *set /* ns */ {},
		quit:       make(chan bool),
	}
}

type set struct {
	inner map[string]string
}

func newSet() *set {
	return &set{
		map[string]string{},
	}
}

func (s *set) put(key string) {
	s.inner[key] = ""
}

func (s *set) exist(key string) bool {
	_, ok := s.inner[key]
	return ok
}

func (s *set) desc() string {
	marshal, _ := json.Marshal(s.inner)
	return string(marshal)
}

func (rbw *roleBindingWatcher) Quit() {
	rbw.quit <- true
}

func (rbw *roleBindingWatcher) Prepare() error {
	c, err := clientcmd.RESTConfigFromKubeConfig([]byte(rbw.kubeConfig))
	if err != nil {
		return err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}

	// create the secret watcher
	rbWatcher := cache.NewListWatchFromClient(
		clientset.RbacV1().RESTClient(), "rolebindings", "",
		fields.OneTermEqualSelector("metadata.name", service.NocalhostDefaultRoleBinding),
	)

	controller := watcher.NewController(rbw, rbWatcher, &rbacv1.RoleBinding{})

	rbw.watchController = controller
	return nil
}

// this method will block until error occur
func (rbw *roleBindingWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go rbw.watchController.Run(1, stop)
	<-rbw.quit
}
