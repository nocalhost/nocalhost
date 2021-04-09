package appmeta_manager

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"nocalhost/internal/nhctl/appmeta"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"sync"
	"time"
)

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
	asw      *applicationSecretWatcher
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller, asw *applicationSecretWatcher) *Controller {
	return &Controller{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
		asw:      asw,
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
		if secret, ok := obj.(*v1.Secret); ok {

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

func (c *Controller) join(secret *v1.Secret) error {
	devMetaBefore := appmeta.ApplicationDevMeta{}
	devMetaCurrent := appmeta.ApplicationDevMeta{}

	asw := c.asw
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
		EventPush(&ApplicationEventPack{
			Event:      event,
			Ns:         asw.ns,
			AppName:    appName,
			KubeConfig: asw.kubeConfig,
		})
	}

	return nil
}

func (c *Controller) left(appName string) {
	devMetaBefore := appmeta.ApplicationDevMeta{}
	devMetaCurrent := appmeta.ApplicationDevMeta{}

	asw := c.asw
	asw.lock.Lock()
	defer asw.lock.Unlock()

	if before, ok := asw.applicationMetas[appName]; ok {
		devMetaBefore = before.GetApplicationDevMeta()
	}
	delete(asw.applicationMetas, appName)

	for _, event := range *devMetaBefore.Events(devMetaCurrent) {
		EventPush(&ApplicationEventPack{
			Event:      event,
			Ns:         asw.ns,
			AppName:    appName,
			KubeConfig: asw.kubeConfig,
		})
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
		log.Infof("Error while resolving %v in Ns %s: %v", key, c.asw.ns, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	log.Error("Dropping resolving %q in Ns %s out of the queue: %v", key, c.asw.ns, err)
}

// Run begins watching and syncing.
func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	log.Infof("Starting Secret watcher with Ns %s", c.asw.ns)

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
	log.Info("Stop Secret watcher with Ns %s", c.asw.ns)
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func NewApplicationSecretWatcher(kubeConfig string, ns string) *applicationSecretWatcher {
	return &applicationSecretWatcher{
		kubeConfig:       kubeConfig,
		ns:               ns,
		applicationMetas: map[string]*appmeta.ApplicationMeta{},
		quit:             make(chan bool),
	}
}

type applicationSecretWatcher struct {
	// todo recreate ASW if kubeConfig changed
	kubeConfig string
	ns         string

	applicationMetas map[string]*appmeta.ApplicationMeta
	lock             sync.Mutex
	quit             chan bool

	watchController *Controller
}

func (asw *applicationSecretWatcher) GetApplicationMetas() (result []*appmeta.ApplicationMeta) {
	for _, meta := range asw.applicationMetas {
		result = append(result, meta)
	}
	return
}

// deep copy to prevent other func change the application meta
func (asw *applicationSecretWatcher) GetApplicationMeta(application string) *appmeta.ApplicationMeta {
	meta := asw.applicationMetas[application]
	if meta != nil {
		copyMeta := meta
		return copyMeta
	} else {
		return &appmeta.ApplicationMeta{
			ApplicationState:   appmeta.UNINSTALLED,
			Ns:                 asw.ns,
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
	c, err := clientcmd.RESTConfigFromKubeConfig([]byte(asw.kubeConfig))
	if err != nil {
		return err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}

	// create the secret watcher
	secretWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "secrets", asw.ns, fields.OneTermEqualSelector("type", appmeta.SecretType))

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the secret key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the Secret than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(secretWatcher, &v1.Secret{}, 0, cache.ResourceEventHandlerFuncs{
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

	controller := NewController(queue, indexer, informer, asw)

	// first get all nocalhost secrets for initial
	list, err := clientset.CoreV1().Secrets(asw.ns).List(context.TODO(),
		metav1.ListOptions{FieldSelector: "type=" + appmeta.SecretType},
	)
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		if err := controller.join(&item); err != nil {
			return err
		}
	}

	asw.watchController = controller
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
