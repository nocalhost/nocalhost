/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cm

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/glog"
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
		return cmw.join(cm.Namespace, appName, cm)
	} else {
		errInfo := fmt.Sprintf("Fetching cm with key %s but could not cast to cm: %v", key, obj)
		glog.Error(errInfo)
		return fmt.Errorf(errInfo)
	}
}

func (cmw *CmWatcher) Delete(key string) error {
	var namespace string
	var app string

	// if key name has namespace/xxx prefix, prefix is the namespace
	if idx := strings.Index(key, "/"); idx > 0 {
		if len(key) > idx+1 {
			namespace = key[:idx]
			cmName := key[idx+1:]

			app = appmeta.GetAppNameFromConfigMapName(cmName)
		}
	}

	if app != "" && namespace != "" {
		cmw.left(namespace, app)
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

func (cmw *CmWatcher) join(ns, app string, cm *corev1.ConfigMap) error {
	cmw.cache.Store(fmt.Sprintf("%s/%s", ns, app), &cm)
	return nil
}

func (cmw *CmWatcher) left(ns, app string) {
	cmw.cache.Delete(fmt.Sprintf("%s/%s", ns, app))
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

	// create the service account watcher
	saWatcher := cache.NewFilteredListWatchFromClient(
		cmw.clientset.CoreV1().RESTClient(), "configmaps", namespace,
		func(options *metav1.ListOptions) {
			options.LabelSelector = kblabels.Set{
				_const.NocalhostCmLabelKey: _const.NocalhostCmLabelValue,
			}.AsSelector().String()
		},
	)

	controller := watcher.NewController(cmw, saWatcher, &corev1.ServiceAccount{})

	listOpt := metav1.ListOptions{}
	listOpt.LabelSelector = kblabels.Set{
		_const.NocalhostCmLabelKey: _const.NocalhostCmLabelValue,
	}.AsSelector().String()

	if list, err := cmw.clientset.CoreV1().ConfigMaps(namespace).List(
		context.TODO(), listOpt,
	); err == nil {
		for _, item := range list.Items {
			if err := cmw.CreateOrUpdate(item.Name, item); err != nil {
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
	if load, ok := cmw.cache.Load(application); ok {
		if cm, ok := load.(corev1.ConfigMap); ok {
			cfgStr := cm.Data[appmeta.CmConfigKey]
			if cfgStr == "" {
				return nil, nil, errors.New(
					fmt.Sprintf(
						"Nocalhost cm %s-%s found, but do not contain %s key in field [Data]",
						cmw.namespace, application, appmeta.CmConfigKey,
					),
				)
			}

			appConfig, svcCfg, err := app.LoadSvcCfgFromStrIfValid(
				cfgStr, svcName,
				base.SvcTypeOf(strings.ToLower(svcType)),
			)
			if err != nil {
				return nil, nil, err
			}
			return appConfig, svcCfg, nil
		}
	}

	return nil, nil, errors.New(
		fmt.Sprintf(
			"Nocalhost cm %s-%s not found",
			cmw.namespace, application,
		),
	)
}

// this method will block until error occur
func (cmw *CmWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go cmw.watchController.Run(1, stop)
	<-cmw.quit
}
