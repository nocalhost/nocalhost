package daemon_handler

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
	"os/exec"
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
			if v.HasKey(resource) {
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
	health          HealthEnum
}

func (c ConnectInfo) toKey() string {
	return generateKey(c.kubeconfigBytes, c.namespace)
}

func (c ConnectInfo) Status() string {
	return c.health.String()
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
				store, _ := h.reverseInfo.LoadOrStore(h.toKey(), &name{
					kubeconfigBytes: h.kubeconfigBytes,
					namespace:       h.namespace,
					resources:       status.Reverse,
				})
				for k, v := range status.Reverse.ele {
					if iv, f := store.(*name).resources.ele[k]; f {
						for _, info := range v.List() {
							if !iv.HasKey(info.string) {
								iv.DeleteByKeys(info.string)
							}
						}
					} else {
						delete(store.(*name).resources.ele, k)
					}
				}
				h.reverseInfoLock.Unlock()
			}
		} else {
			// needs to notify sudo daemon to update connect namespace
			fmt.Println("needs to notify sudo daemon to update connect namespace, this should not to be happen")
			ReleaseWatcher(h.kubeconfigBytes, h.namespace)
			if bytes, err := util.GetClientSetByKubeconfigBytes(h.kubeconfigBytes); err == nil {
				_ = UpdateConnect(bytes, h.namespace, func(list sets.String, item string) {
					list.Delete(item)
				})
				if value, found := GetReverseInfo().LoadAndDelete(h.toKey()); found {
					for _, s := range value.(*name).resources.GetBelongToMeResources().KeySet() {
						_ = UpdateReverseConfigMap(bytes, h.namespace, s,
							func(r *ReverseTotal, record ReverseRecord) {
								r.RemoveRecord(record)
							})
					}
				}
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
				store, _ := h.reverseInfo.LoadOrStore(h.toKey(), &name{
					kubeconfigBytes: h.kubeconfigBytes,
					namespace:       h.namespace,
					resources:       newStatus.Reverse,
				})
				for k, v := range newStatus.Reverse.ele {
					if iv, f := store.(*name).resources.ele[k]; f {
						for _, info := range v.List() {
							if !iv.HasKey(info.string) {
								iv.DeleteByKeys(info.string)
							}
						}
					} else {
						delete(store.(*name).resources.ele, k)
					}
				}
				h.reverseInfoLock.Unlock()
			}
		} else {
			// needs to notify sudo daemon to update connect namespace
			fmt.Println("needs to notify sudo daemon to update connect namespace, this should not to be happen")
			ReleaseWatcher(h.kubeconfigBytes, h.namespace)
			if bytes, err := util.GetClientSetByKubeconfigBytes(h.kubeconfigBytes); err == nil {
				_ = UpdateConnect(bytes, h.namespace, func(list sets.String, item string) {
					list.Delete(item)
				})
				if value, found := GetReverseInfo().LoadAndDelete(h.toKey()); found {
					for _, s := range value.(*name).resources.GetBelongToMeResources().KeySet() {
						_ = UpdateReverseConfigMap(bytes, h.namespace, s,
							func(r *ReverseTotal, record ReverseRecord) {
								r.RemoveRecord(record)
							})
					}
				}
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

// connection healthy and reverse healthy
func init() {
	go func() {
		tick := time.Tick(time.Second * 5)
		for {
			select {
			case <-tick:
				func() {
					defer func() {
						recover()
					}()
					go checkConnect()
					checkReverse()
				}()
			}
		}
	}()
}

func checkConnect() {
	if !connectInfo.IsEmpty() {
		cancel, cancelFunc := context.WithTimeout(context.TODO(), time.Second*5)
		defer cancelFunc()
		cmd := exec.CommandContext(cancel, "ping", "-c", "4", util.IpRange.String())
		_ = cmd.Run()
		if cmd.ProcessState.Success() {
			connectInfo.health = Healthy
		} else {
			connectInfo.health = UnHealthy
		}
	}
}

func checkReverse() {
	if connectInfo.IsEmpty() {
		return
	}
	GetReverseInfo().Range(func(key, value interface{}) bool {
		if !bytes.Equal(value.(*name).kubeconfigBytes, connectInfo.kubeconfigBytes) {
			return true
		}
		path := nocalhost.GetOrGenKubeConfigPath(string(value.(*name).kubeconfigBytes))
		connect := &pkg.ConnectOptions{
			KubeconfigPath: path,
			Namespace:      value.(*name).namespace,
			Workloads:      []string{},
		}
		if err := connect.InitClient(context.TODO()); err != nil {
			return true
		}
		if err := connect.Prepare(context.TODO()); err != nil {
			return true
		}
		value.(*name).resources.GetBelongToMeResources().ForEach(func(k string, v *resourceInfo) {
			go func(k string, info *resourceInfo) {
				connect.Workloads = []string{k}
				if _, err := connect.Shell(context.TODO()); err != nil {
					info.Health = UnHealthy
					fmt.Printf("resource %s is UnHealthy\n", info.string)
				} else {
					info.Health = Healthy
					fmt.Printf("resource %s is health\n", info.string)
				}
			}(k, v)
		})
		return true
	})
}
