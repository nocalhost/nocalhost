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
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"sync"
	"time"
)

type Controller struct {
	indexer cache.Indexer
	queue workqueue.RateLimitingInterface
	informer cache.Controller
	rbw *roleBindingWatcher
}

func NewController(
	queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller, rbw *roleBindingWatcher,
) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		rbw:      rbw,
	}
}

func (c *Controller) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two secrets with the same key are never processed in
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
		log.Errorf("Fetching secret with key %s from store failed with %v", key, err)
		return err
	}

	var appName string
	if exists {
		if secret, ok := obj.(*rbacv1.RoleBinding); ok {

			return c.join(secret)
		} else {
			errInfo := fmt.Sprintf("Fetching secret with key %s but could not cast to secret: %v", key, obj)
			log.Error(errInfo)
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

func (c *Controller) join(rb *rbacv1.RoleBinding) error {
	c.rbw.lock.Lock()
	defer c.rbw.lock.Unlock()

	serviceAccounts := newSet()

	for _, subject := range rb.Subjects {
		serviceAccounts.put(subject.Name)
	}
	c.rbw.ownNs[rb.Namespace] = serviceAccounts
	return nil
}

func (c *Controller) left(rbName string) {
	c.rbw.lock.Lock()
	defer c.rbw.lock.Unlock()

	log.Info("left:" + rbName)
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
		log.Infof("Error while resolving %v %s: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Error("Dropping resolving %q out of the queue: %v", key, err)
}

// Run begins watching and syncing.
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Info("Starting RoleBinding watcher")

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
	log.Info("Stop RoleBinding watcher...")
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
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

type roleBindingWatcher struct {
	// todo recreate RBW if kubeConfig changed
	kubeConfig string

	ownNs map[string] /* ns */ *set /* serviceAccount */
	lock  sync.Mutex
	quit  chan bool

	watchController *Controller
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

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the secret key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(
		rbWatcher, &rbacv1.RoleBinding{}, 0, cache.ResourceEventHandlerFuncs{
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
		}, cache.Indexers{},
	)

	controller := NewController(queue, indexer, informer, rbw)

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
