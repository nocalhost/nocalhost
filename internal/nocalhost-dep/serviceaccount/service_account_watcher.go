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
	"context"
	"fmt"
	"github.com/golang/glog"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"strings"
	"sync"
	"time"
)

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	saw      *ServiceAccountWatcher
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller, saw *ServiceAccountWatcher) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		saw:      saw,
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two sas with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.updateApplicationMeta(key.(string))

	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

func (c *Controller) updateApplicationMeta(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching service account with key %s from store failed with %v", key, err)
		return err
	}

	var appName string
	if exists {
		if sa, ok := obj.(*corev1.ServiceAccount); ok {
			return c.join(sa)
		} else {
			errInfo := fmt.Sprintf("Fetching service account with key %s but could not cast to sa: %v", key, obj)
			glog.Error(errInfo)
			return fmt.Errorf(errInfo)
		}
	} else {
		appName, err = appmeta.GetApplicationName(key)
		if err != nil {
			return err
		}

		c.left(appName)
	}

	return nil
}

func (c *Controller) join(sa *corev1.ServiceAccount) error {
	for key, _ := range sa.Labels {
		if key == clientgo.NocalhostLabel {
			isClusterAdmin, _ := c.saw.isClusterAdmin(sa)
			c.saw.cache.record(string(sa.UID), isClusterAdmin, sa.Name)
			glog.Infof("ServiceAccountCache: refresh nocalhost sa in ns: %s, is cluster admin: %t", sa.Namespace, isClusterAdmin)
			return nil
		}
	}
	return nil
}

func (c *Controller) left(saName string) {
	if idx := strings.Index(saName, "/"); idx > 0 {
		if len(saName) > idx+1 {
			sa := saName[idx+1:]
			glog.Infof("ServiceAccountCache: remove nocalhost sa in ns: %s", saName[:idx])
			c.saw.cache.removeByServiceAccountName(sa)
		}
	}
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.Infof("Error while resolving %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Errorf("Dropping resolving %v out of the queue: %v", key, err)
}

// Run begins watching and syncing.
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	glog.Info("Starting RoleBinding watcher")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stop RoleBinding watcher...")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func NewServiceAccountWatcher(clientset *kubernetes.Clientset) *ServiceAccountWatcher {
	return &ServiceAccountWatcher{
		clientset: clientset,
		cache:     newSet(),
		quit:      make(chan bool),
	}
}

type set struct {
	inner  map[string] /* UID */ bool              /* is cluster admin */
	helper map[string] /* serviceAccount */ string /* UID */
	lock   sync.Mutex
}

func newSet() *set {
	return &set{
		map[string] /* UID */ bool /* is cluster admin */ {},
		map[string] /* serviceAccount */ string /* UID */ {},
		sync.Mutex{},
	}
}

func (s *set) record(key string, isClusterAdmin bool, saName string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.inner[key] = isClusterAdmin
	s.helper[saName] = key
}

func (s *set) removeByServiceAccountName(saName string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if uid, ok := s.helper[saName]; ok {
		delete(s.inner, uid)
		delete(s.helper, saName)
	}
}

type ServiceAccountWatcher struct {
	clientset *kubernetes.Clientset

	cache *set /* serviceAccount */
	lock  sync.Mutex
	quit  chan bool

	watchController *Controller
}

func (saw *ServiceAccountWatcher) isClusterAdmin(sa *corev1.ServiceAccount) (bool, error) {
	if len(sa.Secrets) == 0 {
		return false, nil
	}

	secret, err := saw.clientset.CoreV1().Secrets(sa.Namespace).Get(context.TODO(), sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		glog.Error(err)
		return false, err
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Error(err)
		return false, err
	}

	KubeConfigYaml, err, _ := setupcluster.NewDevKubeConfigReader(secret, config.Host, sa.Namespace).GetCA().GetToken().AssembleDevKubeConfig().ToYamlString()
	if err != nil {
		glog.Error(err)
		return false, err
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(KubeConfigYaml))
	if err != nil {
		glog.Error(err)
		return false, nil
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Error(err)
		return false, nil
	}

	arg := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: "*",
				Group:     "*",
				Verb:      "*",
				Name:      "*",
				Version:   "*",
				Resource:  "*",
			},
		},
	}

	response, err := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(context.TODO(), arg, metav1.CreateOptions{})
	if err != nil {
		glog.Error(err)
		return false, err
	}
	return response.Status.Allowed, nil
}

func (saw *ServiceAccountWatcher) IsClusterAdmin(uid string) *bool {
	admin, ok := saw.cache.inner[uid]
	if ok {
		return &admin
	} else {
		return nil
	}
}

func (saw *ServiceAccountWatcher) Quit() {
	saw.quit <- true
}

func (saw *ServiceAccountWatcher) Prepare() error {
	// create the service account watcher
	saWatcher := cache.NewListWatchFromClient(saw.clientset.CoreV1().RESTClient(), "serviceaccounts", "default", fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the service account key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the service account than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(saWatcher, &corev1.ServiceAccount{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewController(queue, indexer, informer, saw)

	if list, err := saw.clientset.CoreV1().ServiceAccounts("default").List(context.TODO(), metav1.ListOptions{}); err == nil {
		for _, item := range list.Items {
			for key, _ := range item.Labels {
				if key == clientgo.NocalhostLabel {
					isClusterAdmin, _ := saw.isClusterAdmin(&item)
					saw.cache.record(string(item.UID), isClusterAdmin, item.Name)
					glog.Infof("ServiceAccountCache: refresh nocalhost sa in ns: %s, is cluster admin: %t", item.Namespace, isClusterAdmin)
				}
			}
		}
	}

	saw.watchController = controller
	return nil
}

// this method will block until error occur
func (saw *ServiceAccountWatcher) Watch() {
	stop := make(chan struct{})
	defer close(stop)
	go saw.watchController.Run(1, stop)
	<-saw.quit
}
