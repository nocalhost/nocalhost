package daemon_handler

import (
	"crypto/sha1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/vpn/util"
	"sync"
	"time"
)

var watchers = map[string]*ConfigMapWatcher{}
var watchersLock = sync.Mutex{}

func ReleaseWatcher(kubeconfigBytes []byte, namespace string) {
	watchersLock.Lock()
	defer watchersLock.Unlock()
	key := generateKey(kubeconfigBytes, namespace)
	if v, ok := watchers[key]; ok && v != nil {
		v.Stop()
	}
}

func GetOrGenerateConfigMapWatcher(kubeconfigBytes []byte, namespace string, getter cache.Getter) *ConfigMapWatcher {
	watchersLock.Lock()
	key := generateKey(kubeconfigBytes, namespace)
	if v, ok := watchers[key]; ok && v != nil {
		watchersLock.Unlock()
		return v
	} else {
		watcher := NewConfigMapWatcher(kubeconfigBytes, namespace, getter)
		watchers[key] = watcher
		watchersLock.Unlock()
		watcher.Start()
		return watcher
	}
}

var connectInfo = &ConnectInfo{}
var reverseLock = sync.Mutex{}
var reverseInfo = sets.NewString()

type ConnectInfo struct {
	kubeconfigBytes []byte
	namespace       string
}

func Reverse(kubeconfigBytes []byte, namespace string, resource string) bool {
	reverseLock.Lock()
	defer reverseLock.Unlock()
	if (string(connectInfo.kubeconfigBytes) == string(kubeconfigBytes)) &&
		(connectInfo.namespace == namespace) &&
		reverseInfo.Has(resource) {
		return true
	}
	return false
}

func (c *ConnectInfo) IsEmpty() bool {
	return c == nil || len(c.namespace) == 0 || len(c.kubeconfigBytes) == 0
}

type ConfigMapWatcher struct {
	informer        cache.SharedInformer
	kubeconfigBytes []byte
	namespace       string
	stopChan        chan struct{}
}

func NewConfigMapWatcher(kubeconfigBytes []byte, namespace string, getter cache.Getter) *ConfigMapWatcher {
	informer := cache.NewSharedInformer(
		cache.NewListWatchFromClient(
			getter,
			"configmaps",
			namespace,
			fields.OneTermEqualSelector("metadata.name", util.TrafficManager),
		),
		&corev1.ConfigMap{},
		time.Second*5,
	)
	informer.AddEventHandler(&resourceHandler{
		namespace:       namespace,
		kubeconfigBytes: kubeconfigBytes,
		lock:            &reverseLock,
		reverseInfo:     reverseInfo,
	})
	return &ConfigMapWatcher{
		informer:        informer,
		kubeconfigBytes: kubeconfigBytes,
		namespace:       namespace,
		stopChan:        make(chan struct{}, 1),
	}
}

func (w *ConfigMapWatcher) Start() {
	go w.informer.Run(w.stopChan)
	for !w.informer.HasSynced() {
	}
}

func (w *ConfigMapWatcher) Stop() {
	w.stopChan <- struct{}{}
}

type resourceHandler struct {
	namespace       string
	kubeconfigBytes []byte
	lock            *sync.Mutex
	connectInfo     *ConnectInfo
	reverseInfo     sets.String
}

func (h *resourceHandler) OnAdd(obj interface{}) {
	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() {
		go h.notifyDaemonToConnect()

		connectInfo.namespace = h.namespace
		connectInfo.kubeconfigBytes = h.kubeconfigBytes
		if len(connectInfo.namespace) != 0 {
			lock.Lock()
			h.reverseInfo.Insert(status.Reverse.GetBelongToMeResources().List()...)
			lock.Unlock()
		}
	}
}

func (h *resourceHandler) OnUpdate(oldObj, newObj interface{}) {
	configMap := newObj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() {
		if generateKey(connectInfo.kubeconfigBytes, connectInfo.namespace) !=
			generateKey(h.kubeconfigBytes, h.namespace) && !connectInfo.IsEmpty() {
			// needs to notify sudo daemon to update connect namespace
			fmt.Println("needs to notify sudo daemon to update connect namespace, this should not to be happen")
		}
		go h.notifyDaemonToConnect()
		connectInfo.namespace = h.namespace
		connectInfo.kubeconfigBytes = h.kubeconfigBytes
		if len(connectInfo.namespace) != 0 {
			lock.Lock()
			h.reverseInfo.Insert(status.Reverse.GetBelongToMeResources().List()...)
			lock.Unlock()
		}
	}
}

func (h *resourceHandler) OnDelete(obj interface{}) {
	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() {
		connectInfo.namespace = ""
		connectInfo.kubeconfigBytes = []byte{}
		h.reverseInfo.Delete(status.Reverse.GetBelongToMeResources().List()...)
	}
}

func (h *resourceHandler) notifyDaemonToConnect() error {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return err
	}
	path := nocalhost.GetOrGenKubeConfigPath(string(h.kubeconfigBytes))
	if _, err = client.SendVPNOperateCommand(path, h.namespace, command.Connect, ""); err != nil {
		return err
	}
	return nil
}

func generateKey(kubeconfigBytes []byte, namespace string) string {
	h := sha1.New()
	h.Write(kubeconfigBytes)
	return string(h.Sum([]byte(namespace)))
}
