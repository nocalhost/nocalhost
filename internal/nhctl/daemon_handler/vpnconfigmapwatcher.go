package daemon_handler

import (
	"context"
	"crypto/sha1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/vpn/util"
	"sync"
	"time"
)

var watchers = map[string]*ConfigMapWatcher{}
var watchersLock = &sync.Mutex{}

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
	defer watchersLock.Unlock()
	k := generateKey(kubeconfigBytes, namespace)
	if v, ok := watchers[k]; ok && v != nil {
		return v
	} else {
		if getter == nil {
			if clientset, err := util.GetClientSetByKubeconfigBytes(kubeconfigBytes); err != nil {
				return nil
			} else {
				getter = clientset.CoreV1().RESTClient()
			}
		}
		if watcher := NewConfigMapWatcher(kubeconfigBytes, namespace, getter); watcher != nil {
			watchers[k] = watcher
			watcher.Start()
			return watcher
		}
		return nil
	}
}

var connectInfo = &ConnectInfo{}
var reverseInfoLock = &sync.Mutex{}

// kubeconfig+ns --> name
var reverseInfo = &sync.Map{}

type name struct {
	kubeconfigBytes []byte
	namespace       string
	// mac --> resources
	resources *ReverseTotal
}

func (n *name) getMacByResource(resource string) string {
	reverseInfoLock.Lock()
	defer reverseInfoLock.Unlock()
	if load, ok := reverseInfo.Load(generateKey(n.kubeconfigBytes, n.namespace)); ok {
		for k, v := range load.(*name).resources.ele {
			if v.Has(resource) {
				return k
			}
		}
	}
	return ""
}

func (n *name) isReversingByMe(resource string) bool {
	if a := n.getMacByResource(resource); len(a) != 0 {
		if util.GetMacAddress().String() == a {
			return true
		}
	}
	return false
}

func (n *name) isReversingByOther(resource string) bool {
	if n.isReversingByMe(resource) {
		return false
	}
	if a := n.getMacByResource(resource); len(a) != 0 {
		if util.GetMacAddress().String() != a {
			return true
		}
	}
	return false
}

func GetReverseInfo() *sync.Map {
	return reverseInfo
}

type ConnectInfo struct {
	kubeconfigBytes []byte
	namespace       string
	ip              string
}

func (c *ConnectInfo) cleanup() {
	c.namespace = ""
	c.kubeconfigBytes = []byte{}
	c.ip = ""
}

func (c *ConnectInfo) IsEmpty() bool {
	return c == nil || len(c.namespace) == 0 || len(c.kubeconfigBytes) == 0
}

func (c *ConnectInfo) IsSame(kubeconfigBytes []byte, namespace string) bool {
	return string(kubeconfigBytes) == string(c.kubeconfigBytes) && namespace == c.namespace
}

func (c *ConnectInfo) IsSameCluster(kubeconfigBytes []byte) bool {
	return string(kubeconfigBytes) == string(c.kubeconfigBytes)
}

func (c *ConnectInfo) getIPIfIsMe(kubeconfigBytes []byte, namespace string) (ip string) {
	if c.IsSame(kubeconfigBytes, namespace) {
		ip = c.ip
	}
	return
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
		reverseInfoLock: reverseInfoLock,
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
	reverseInfoLock *sync.Mutex
	reverseInfo     *sync.Map
	backoffTime     time.Time
}

func (h *resourceHandler) toKey() string {
	return generateKey(h.kubeconfigBytes, h.namespace)
}

func (h *resourceHandler) OnAdd(obj interface{}) {
	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() {
		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) || connectInfo.IsEmpty() {
			go h.notifySudoDaemonToConnect()

			connectInfo.namespace = h.namespace
			connectInfo.kubeconfigBytes = h.kubeconfigBytes
			connectInfo.ip = status.MacToIP[util.GetMacAddress().String()]
			if len(connectInfo.namespace) != 0 {
				h.reverseInfoLock.Lock()
				h.reverseInfo.Store(h.toKey(), &name{
					kubeconfigBytes: h.kubeconfigBytes,
					namespace:       h.namespace,
					resources:       status.Reverse,
				})
				h.reverseInfoLock.Unlock()
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
			connectInfo.ip = newStatus.MacToIP[util.GetMacAddress().String()]
			if len(connectInfo.namespace) != 0 {
				h.reverseInfoLock.Lock()
				h.reverseInfo.Store(h.toKey(), &name{
					kubeconfigBytes: h.kubeconfigBytes,
					namespace:       h.namespace,
					resources:       newStatus.Reverse,
				})
				h.reverseInfoLock.Unlock()
			}
		} else {
			// needs to notify sudo daemon to update connect namespace
			fmt.Println("needs to notify sudo daemon to update connect namespace, this should not to be happen")
			ReleaseWatcher(h.kubeconfigBytes, h.namespace)
			if bytes, err := util.GetClientSetByKubeconfigBytes(h.kubeconfigBytes); err == nil {
				_ = bytes.CoreV1().
					ConfigMaps(h.namespace).
					Delete(context.TODO(), util.TrafficManager, v1.DeleteOptions{})
			}
		}
	}
	// if connected --> disconnected
	if oldStatus.Connect.IsConnected() && !newStatus.Connect.IsConnected() {
		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
			connectInfo.cleanup()
		}
		if h.backoffTime.Unix() < time.Now().Unix() {
			go h.notifySudoDaemonToDisConnect()
			h.backoffTime = time.Now().Add(time.Second * 5)
		}
	}
}

func (h *resourceHandler) OnDelete(obj interface{}) {
	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() && connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
		connectInfo.cleanup()
		h.reverseInfoLock.Lock()
		h.reverseInfo.Delete(h.toKey())
		h.reverseInfoLock.Unlock()
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
	//go func(info *ConnectInfo) {
	//	if util.IsAdmin() {
	//		return
	//	}
	//	for {
	//		var kubeConfigHost, namespace string
	//		if info != nil {
	//			namespace = info.namespace
	//			kubeConfig, _ := clientcmd.RESTConfigFromKubeConfig(info.kubeconfigBytes)
	//			if kubeConfig != nil {
	//				kubeConfigHost = kubeConfig.Host
	//			}
	//		}
	//		fmt.Printf("namespace: %s, kubeconfig: %s\n", namespace, kubeConfigHost)
	//		<-time.Tick(time.Second * 5)
	//	}
	//}(connectInfo)
}
