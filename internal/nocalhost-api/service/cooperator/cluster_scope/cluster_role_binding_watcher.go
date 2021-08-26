/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_scope

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
	"nocalhost/internal/nocalhost-api/service/cooperator/util"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strings"
	"sync"
)

var splitter = ":"
var constCooperator = "cooperator"
var constViewer = "viewer"

type clusterRoleBindingWatcher struct {
	// todo recreate RBW if kubeConfig changed
	kubeConfig string

	cache sync.Map // map[sa]*saShareContainer

	lock sync.RWMutex
	quit chan bool

	watchController *watcher.Controller
}

type saShareContainer struct {

	// in normal case, ownerValid must be true
	// while viewerSas or cooperatorSas is not empty
	ownerValid                      bool
	viewerSas/* ns */ *util.Set     /* serviceAccount */
	cooperatorSas/* ns */ *util.Set /* serviceAccount */
	lock                            sync.RWMutex
}

func newSaContainer() *saShareContainer {
	return &saShareContainer{
		viewerSas:     &util.Set{},
		cooperatorSas: &util.Set{},
	}
}

func (ssc *saShareContainer) updateViewerSas(s *util.Set) {
	ssc.lock.Lock()
	defer ssc.lock.Unlock()

	ssc.viewerSas = s
}

func (ssc *saShareContainer) updateCooperSas(s *util.Set) {
	ssc.lock.Lock()
	defer ssc.lock.Unlock()

	ssc.cooperatorSas = s
}

func (crbw *clusterRoleBindingWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if crb, ok := obj.(*rbacv1.ClusterRoleBinding); ok {
		return crbw.join(crb)
	} else {
		errInfo := fmt.Sprintf("Fetching clusterRoleBinding with key %s but could not cast to clusterRoleBinding: %v", key, obj)
		log.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (crbw *clusterRoleBindingWatcher) Delete(key string) error {
	// todo
	log.Infof("cluster role binding delete: %s", key)
	return nil
}

func (crbw *clusterRoleBindingWatcher) WatcherInfo() string {
	return fmt.Sprintf("'ClusterRoleBinding'")
}

// nocalhost rbac cluster role binding naming rules is
// nocalhost:{serviceAccount}:cooperator
// nocalhost:{serviceAccount}:viewer
//
// serviceAccount is the owner's service account
// if owner has been unauthorization from cluster dev space
// we should delete corresponding cluster role binding
func (crbw *clusterRoleBindingWatcher) join(crb *rbacv1.ClusterRoleBinding) error {
	crbw.lock.Lock()
	defer crbw.lock.Unlock()

	crbName := crb.Name

	// parse the owner first
	if crbName == _const.NocalhostDefaultRoleBinding {
		existSubject := util.Set{}
		for _, subject := range crb.Subjects {
			val, _ := crbw.cache.LoadOrStore(
				subject.Name,
				// todo: this will be create every time, go api is ridiculous
				newSaContainer(),
			)
			ssc := val.(*saShareContainer)
			ssc.ownerValid = true
			existSubject.Put(subject.Name)
		}

		// remove those sa do not as cluster admin
		crbw.cache.Range(
			func(key, val interface{}) bool {
				if !existSubject.Exist(key.(string)) {
					ssc := val.(*saShareContainer)
					ssc.ownerValid = false
				}
				return true
			})

		return nil
	}

	// then if not the devSpace owner, use nocalhost:{serviceAccount}:xxx
	// to parse the sharer
	split := strings.Split(crbName, splitter)
	if len(split) != 3 || len(split) == 0 ||
		split[0] != "nocalhost" ||
		(split[2] != "cooperator" && split[2] != "viewer") {
		return nil
	}

	ownerSa := split[1]
	spaceOwnType := split[2]

	switch spaceOwnType {
	case constCooperator:

		s := &util.Set{}
		for _, subject := range crb.Subjects {
			s.Put(subject.Name)
		}

		val, _ := crbw.cache.LoadOrStore(
			ownerSa,
			// todo: this will be create every time, go api is ridiculous
			newSaContainer(),
		)
		ssc := val.(*saShareContainer)

		ssc.updateCooperSas(s)
	case constViewer:

		s := &util.Set{}
		for _, subject := range crb.Subjects {
			s.Put(subject.Name)
		}

		val, _ := crbw.cache.LoadOrStore(
			ownerSa,
			// todo: this will be create every time, go api is ridiculous
			newSaContainer(),
		)
		ssc := val.(*saShareContainer)

		ssc.updateViewerSas(s)
	default:
		log.Warnf(
			"sa %s found with nocalhost's label, but not one of %s or %s",
			crb.Name, "cluster:"+_const.NocalhostViewerRoleBinding, "cluster:"+_const.NocalhostDefaultRoleBinding,
		)
	}

	return nil
}

func NewClusterRoleBindingWatcher(kubeConfig string) *clusterRoleBindingWatcher {
	return &clusterRoleBindingWatcher{
		kubeConfig: kubeConfig,
		cache:      sync.Map{},
		lock:       sync.RWMutex{},
		quit:       make(chan bool),
	}
}

func (crbw *clusterRoleBindingWatcher) Quit() {
	crbw.quit <- true
}

func (crbw *clusterRoleBindingWatcher) Prepare() error {
	c, err := clientcmd.RESTConfigFromKubeConfig([]byte(crbw.kubeConfig))
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
		clientset.RbacV1().RESTClient(), "clusterrolebindings", "",
		func(options *metav1.ListOptions) {
			options.LabelSelector = kblabels.Set{
				_const.NocalhostRoleBindingLabelKey: _const.NocalhostRoleBindingLabelVal,
			}.AsSelector().String()
		},
	)

	controller := watcher.NewController(crbw, rbWatcher, &rbacv1.ClusterRoleBinding{})

	crbw.watchController = controller
	return nil
}

// this method will block until error occur
func (crbw *clusterRoleBindingWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go crbw.watchController.Run(1, stop)
	<-crbw.quit
}
