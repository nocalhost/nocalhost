package daemon_handler

import (
	"context"
	"crypto/sha1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
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
	k := generateKey(kubeconfigBytes, namespace)
	if v, ok := watchers[k]; ok && v != nil {
		v.Stop()
	}
}

func GetOrGenerateConfigMapWatcher(kubeconfigBytes []byte, namespace string, getter cache.Getter) *ConfigMapWatcher {
	watchersLock.Lock()
	k := generateKey(kubeconfigBytes, namespace)
	if v, ok := watchers[k]; ok && v != nil {
		watchersLock.Unlock()
		return v
	} else {
		watcher := NewConfigMapWatcher(kubeconfigBytes, namespace, getter)
		watchers[k] = watcher
		watchersLock.Unlock()
		watcher.Start()
		return watcher
	}
}

var connectInfo = &ConnectInfo{}
var reverseInfoLock = sync.Mutex{}
var reverseInfo = sets.NewString()

func GetReverseInfo() sets.String {
	data := sets.NewString()
	reverseInfoLock.Lock()
	data.Insert(reverseInfo.List()...)
	reverseInfoLock.Unlock()
	return data
}

type ConnectInfo struct {
	kubeconfigBytes []byte
	namespace       string
}

func (c *ConnectInfo) Cleanup() {
	c.namespace = ""
	c.kubeconfigBytes = []byte{}
}

func Reverse(kubeconfigBytes []byte, namespace string, resource string) bool {
	reverseInfoLock.Lock()
	defer reverseInfoLock.Unlock()
	if connectInfo.IsSame(kubeconfigBytes, namespace) && reverseInfo.Has(resource) {
		return true
	}
	return false
}

func (c *ConnectInfo) IsEmpty() bool {
	return c == nil || len(c.namespace) == 0 || len(c.kubeconfigBytes) == 0
}

func (c *ConnectInfo) IsSame(kubeconfigBytes []byte, namespace string) bool {
	return string(kubeconfigBytes) == string(c.kubeconfigBytes) && namespace == c.namespace
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
		lock:            &reverseInfoLock,
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
		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) || connectInfo.IsEmpty() {
			go h.notifySudoDaemonToConnect()

			connectInfo.namespace = h.namespace
			connectInfo.kubeconfigBytes = h.kubeconfigBytes
			if len(connectInfo.namespace) != 0 {
				lock.Lock()
				h.reverseInfo.Insert(status.Reverse.GetBelongToMeResources().List()...)
				lock.Unlock()
			}
		} else {
			ReleaseWatcher(h.kubeconfigBytes, h.namespace)
			if bytes, err := util.GetClientSetByKubeconfigBytes(h.kubeconfigBytes); err == nil {
				_ = bytes.CoreV1().ConfigMaps(h.namespace).
					Delete(context.TODO(), util.TrafficManager, v1.DeleteOptions{})
			}
		}
	}
}

func (h *resourceHandler) OnUpdate(oldObj, newObj interface{}) {
	oldStatus := ToStatus(oldObj.(*corev1.ConfigMap).Data)
	newStatus := ToStatus(newObj.(*corev1.ConfigMap).Data)
	if newStatus.Connect.IsConnected() {
		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) || connectInfo.IsEmpty() {
			go h.notifySudoDaemonToConnect()
			connectInfo.namespace = h.namespace
			connectInfo.kubeconfigBytes = h.kubeconfigBytes
			if len(connectInfo.namespace) != 0 {
				lock.Lock()
				h.reverseInfo.Insert(newStatus.Reverse.GetBelongToMeResources().List()...)
				lock.Unlock()
			}
		} else {
			// needs to notify sudo daemon to update connect namespace
			fmt.Println("needs to notify sudo daemon to update connect namespace, this should not to be happen")
			ReleaseWatcher(h.kubeconfigBytes, h.namespace)
			if bytes, err := util.GetClientSetByKubeconfigBytes(h.kubeconfigBytes); err == nil {
				_ = bytes.CoreV1().ConfigMaps(h.namespace).
					Delete(context.TODO(), util.TrafficManager, v1.DeleteOptions{})
			}
		}
	}
	// if connected --> disconnected
	if oldStatus.Connect.IsConnected() && !newStatus.Connect.IsConnected() {
		go h.notifySudoDaemonToDisConnect()
	}
}

func (h *resourceHandler) OnDelete(obj interface{}) {
	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() && connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
		connectInfo.namespace = ""
		connectInfo.kubeconfigBytes = []byte{}
		h.reverseInfo.Delete(status.Reverse.GetBelongToMeResources().List()...)
	}
}

func (h *resourceHandler) notifySudoDaemonToConnect() error {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return err
	}
	path := nocalhost.GetOrGenKubeConfigPath(string(h.kubeconfigBytes))
	if _, err = client.SendSudoVPNOperateCommand(path, h.namespace, command.Connect, ""); err != nil {
		return err
	}
	return nil
}

func (h *resourceHandler) notifySudoDaemonToDisConnect() error {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return err
	}
	path := nocalhost.GetOrGenKubeConfigPath(string(h.kubeconfigBytes))
	if _, err = client.SendSudoVPNOperateCommand(path, h.namespace, command.DisConnect, ""); err != nil {
		return err
	}
	return nil
}

func generateKey(kubeconfigBytes []byte, namespace string) string {
	h := sha1.New()
	h.Write(kubeconfigBytes)
	return string(h.Sum([]byte(namespace)))
}

func init() {
	go func(info *ConnectInfo) {
		for {
			select {
			case <-time.Tick(time.Second * 5):
				if info != nil {
					kubeConfig, _ := clientcmd.RESTConfigFromKubeConfig(info.kubeconfigBytes)
					if kubeConfig != nil {
						fmt.Printf("namespace: %s, kubeconfig: %s\n",
							info.namespace, kubeConfig.Host)
					}
				}
			}
		}
	}(connectInfo)
}
