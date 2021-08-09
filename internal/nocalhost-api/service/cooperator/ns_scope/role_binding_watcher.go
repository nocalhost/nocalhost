/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ns_scope

import (
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strings"
	"sync"
)

type roleBindingWatcher struct {
	// todo recreate RBW if kubeConfig changed
	kubeConfig string

	adminSas      map[string] /* ns */ *set /* serviceAccount */
	viewerSas     map[string] /* ns */ *set /* serviceAccount */
	cooperatorSas map[string] /* ns */ *set /* serviceAccount */
	lock          sync.RWMutex
	quit          chan bool

	watchController *watcher.Controller
}

func (rbw *roleBindingWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if secret, ok := obj.(*rbacv1.RoleBinding); ok {
		return rbw.join(secret)
	} else {
		errInfo := fmt.Sprintf("Fetching roleBiding with key %s but could not cast to roleBiding: %v", key, obj)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (rbw *roleBindingWatcher) Delete(key string) error {
	var rbName string
	var ns string
	if idx := strings.Index(key, "/"); idx > 0 {
		if len(key) > idx+1 {
			rbName = key[idx+1:]
			ns = key[:idx]

			if ns == "" {
				ns = "default"
			}
		}

		rbw.lock.Lock()
		defer rbw.lock.Unlock()

		if rbName == _const.NocalhostViewerRoleBinding {

			delete(rbw.viewerSas, ns)
		} else if rbName == _const.NocalhostDefaultRoleBinding {

			delete(rbw.adminSas, ns)
		} else if rbName == _const.NocalhostCooperatorRoleBinding {

			delete(rbw.cooperatorSas, ns)
		} else {

			log.Warnf(
				"sa %s found with nocalhost's label, but not one of %s or %s",
				rbName, _const.NocalhostViewerRoleBinding, _const.NocalhostDefaultRoleBinding,
			)
		}

	} else {
		return nil
	}
	return nil
}

func (rbw *roleBindingWatcher) WatcherInfo() string {
	return fmt.Sprintf("'RoleBinding - ns:%s'", rbw.adminSas)
}

func (rbw *roleBindingWatcher) join(rb *rbacv1.RoleBinding) error {
	rbw.lock.Lock()
	defer rbw.lock.Unlock()

	// nocalhost owner use _const.NocalhostViewerRoleBinding as rb
	// and cooperator use _const.NocalhostCooperatorRoleBinding as rb
	// and viewer use _const.NocalhostDefaultRoleBinding as rb
	if rb.Name == _const.NocalhostViewerRoleBinding {

		viewerSas := newSet()
		for _, subject := range rb.Subjects {
			viewerSas.put(subject.Name)
		}
		rbw.viewerSas[rb.Namespace] = viewerSas
	} else if rb.Name == _const.NocalhostDefaultRoleBinding {

		adminSas := newSet()
		for _, subject := range rb.Subjects {
			adminSas.put(subject.Name)
		}
		rbw.adminSas[rb.Namespace] = adminSas
	} else if rb.Name == _const.NocalhostCooperatorRoleBinding {

		cooperSas := newSet()
		for _, subject := range rb.Subjects {
			cooperSas.put(subject.Name)
		}
		rbw.cooperatorSas[rb.Namespace] = cooperSas
	} else {

		log.Warnf(
			"sa %s found with nocalhost's label, but not one of %s or %s",
			rb.Name, _const.NocalhostViewerRoleBinding, _const.NocalhostDefaultRoleBinding,
		)
	}

	return nil
}

func NewRoleBindingWatcher(kubeConfig string) *roleBindingWatcher {
	return &roleBindingWatcher{
		kubeConfig:    kubeConfig,
		adminSas:      map[string] /* serviceAccount */ *set /* ns */ {},
		cooperatorSas: map[string] /* serviceAccount */ *set /* ns */ {},
		viewerSas:     map[string] /* serviceAccount */ *set /* ns */ {},
		quit:          make(chan bool),
	}
}

type set struct {
	inner sync.Map
}

func newSet() *set {
	return &set{
		sync.Map{},
	}
}

func (s *set) toArray() []string {
	result := make([]string, 0)

	s.inner.Range(
		func(key, value interface{}) bool {
			result = append(result, key.(string))
			return true
		},
	)

	return result
}

func (s *set) put(key string) {
	s.inner.Store(key, "")
}

func (s *set) exist(key string) bool {
	_, ok := s.inner.Load(key)
	return ok
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
	rbWatcher := cache.NewFilteredListWatchFromClient(
		clientset.RbacV1().RESTClient(), "rolebindings", "",
		func(options *metav1.ListOptions) {
			options.LabelSelector = kblabels.Set{
				_const.NocalhostRoleBindingLabelKey: _const.NocalhostRoleBindingLabelVal,
			}.AsSelector().String()
		},
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
