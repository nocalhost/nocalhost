package watcher

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"nocalhost/pkg/nhctl/clientgoutils"
	"time"
)

type SimpleWatcher struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	watcher  Watcher
}

// NewSimpleWatcher returns the chan control the informer's live cycle
// you can stop the informer by calling
// quitChan <- whatever
// or block the thread by this quitChan
//
// the quitChan in createOrUpdateFunc, deleteFunc and returns is the same
func NewSimpleWatcher(
	cgu *clientgoutils.ClientGoUtils, resource, labelSelector string,
	createOrUpdate func(key string, object interface{}, quitChan chan struct{}),
	delete func(key string, quitChan chan struct{}),
) chan struct{} {
	gvr := cgu.ResourceFor(resource, false)
	var lofun dynamicinformer.TweakListOptionsFunc = nil
	if labelSelector != "" {
		lofun = func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		}
	}

	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		cgu.GetDynamicClient(),
		time.Second*2,
		cgu.GetNameSpace(),
		lofun,
	)

	stopCh := make(chan struct{})

	informer := dynamicInformerFactory.ForResource(gvr)
	fun := func(key string) {
		if obj, exists, err := informer.Informer().GetIndexer().GetByKey(key); err == nil {
			if exists {
				if createOrUpdate != nil {
					createOrUpdate(key, obj, stopCh)
				}
			} else {
				if delete != nil {
					delete(key, stopCh)
				}
			}
		}
	}

	informer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
					fun(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				if key, err := cache.MetaNamespaceKeyFunc(new); err == nil {
					fun(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
					fun(key)
				}
			},
		},
	)

	go informer.Informer().Run(stopCh)
	return stopCh
}
