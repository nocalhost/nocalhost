/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cm

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/watcher"
	"nocalhost/pkg/nhctl/log"
	"strings"
	"sync"
)

var NOT_FOUND = errors.New("Not found")

type CmWatcher struct {
	namespace string

	clientset *kubernetes.Clientset

	cache sync.Map
	lock  sync.Mutex
	quit  chan bool

	watchController *watcher.Controller
}

func (cmw *CmWatcher) CreateOrUpdate(key string, obj interface{}) error {
	if cm, ok := obj.(*corev1.ConfigMap); ok {
		appName := appmeta.GetAppNameFromConfigMapName(cm.Name)
		if appName == "" {
			log.Info(
				"cm %s is not standard nocalhost "+
					"config (should named with dev.nocalhost.config.${app})", appName,
			)
			return nil
		}
		return cmw.join(appName, cm)
	} else {
		errInfo := fmt.Sprintf("Fetching cm with key %s but could not cast to cm: %v", key, obj)
		glog.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (cmw *CmWatcher) Delete(key string) error {
	var namespace string
	var appName string

	// if key name has namespace/xxx prefix, prefix is the namespace
	if idx := strings.Index(key, "/"); idx > 0 {
		if len(key) > idx+1 {
			namespace = key[:idx]
			cmName := key[idx+1:]

			appName = appmeta.GetAppNameFromConfigMapName(cmName)
		}
	}

	if appName != "" && namespace != "" {
		cmw.left(appName)
	} else {
		log.Info(
			"[Delete Event] cm %s is not standard nocalhost "+
				"config (should named with dev.nocalhost.config.${app})", key,
		)
	}

	return nil
}

func (cmw *CmWatcher) WatcherInfo() string {
	return fmt.Sprintf("'Cm'")
}

func (cmw *CmWatcher) join(app string, cm *corev1.ConfigMap) error {
	cmw.cache.Store(app, cm)
	return nil
}

func (cmw *CmWatcher) left(app string) {
	cmw.cache.Delete(app)
}

func NewCmWatcher(clientset *kubernetes.Clientset) *CmWatcher {
	return &CmWatcher{
		clientset: clientset,
		cache:     sync.Map{},
		quit:      make(chan bool),
	}
}

func (cmw *CmWatcher) Prepare(namespace string) error {
	cmw.namespace = namespace

	//create the service account watcher
	saWatcher := cache.NewFilteredListWatchFromClient(
		cmw.clientset.CoreV1().RESTClient(), "configmaps", namespace,
		func(options *metav1.ListOptions) {
			options.LabelSelector = kblabels.Set{
				_const.NocalhostCmLabelKey: _const.NocalhostCmLabelValue,
			}.AsSelector().String()
		},
	)

	controller := watcher.NewController(cmw, saWatcher, &corev1.ConfigMap{})

	listOpt := metav1.ListOptions{}
	listOpt.LabelSelector = kblabels.Set{
		_const.NocalhostCmLabelKey: _const.NocalhostCmLabelValue,
	}.AsSelector().String()

	if list, err := cmw.clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(), listOpt,
	); err == nil {
		for _, item := range list.Items {
			if err := cmw.CreateOrUpdate(item.Name, &item); err != nil {
				glog.Infof(
					"CmCache: load nocalhost cm config in ns: %s error: %s", item.Namespace,
					err.Error(),
				)
			}
		}
	}

	cmw.watchController = controller
	return nil
}

// GetNocalhostConfig
// CAUTION: appCfg may nil!
func (cmw *CmWatcher) GetNocalhostConfig(application, svcType, svcName string) (
	*profile.NocalHostAppConfigV2, *profile.ServiceConfigV2, error) {

	glog.Infof("App: %s, svcType: %s, svcName: %s", application, svcType, svcName)
	if load, ok := cmw.cache.Load(application); ok {
		if cm, ok := load.(*corev1.ConfigMap); ok {
			cfgStr := cm.Data[appmeta.CmConfigKey]
			if cfgStr == "" {
				return nil, nil, errors.New(
					fmt.Sprintf(
						"Nocalhost cm %s-%s found, but do not contain %s key in field [Data]",
						cmw.namespace, application, appmeta.CmConfigKey,
					),
				)
			}

			return app.LoadSvcCfgFromStrIfValid(
				cfgStr, svcName,
				base.SvcType(strings.ToLower(svcType)),
			)
		}
	}

	return nil, nil, NOT_FOUND
}

// this method will block until error occur
func (cmw *CmWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go cmw.watchController.Run(1, stop)
	<-cmw.quit
}
