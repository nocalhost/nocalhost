package daemon_handler

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/k8sutils"
	"os/exec"
	"runtime/debug"
	"sync"
	"time"
)

var watchers = map[string]*ConfigMapWatcher{}
var watchersLock = &sync.Mutex{}

func ReleaseWatcher(kubeconfigBytes []byte, namespace string) {
	watchersLock.Lock()
	defer watchersLock.Unlock()
	k := util.GenerateKey(kubeconfigBytes, namespace)
	if v, ok := watchers[k]; ok && v != nil {
		v.Stop()
		delete(watchers, k)
	}
}

func GetOrGenerateConfigMapWatcher(kubeconfigBytes []byte, namespace string, getter cache.Getter) *ConfigMapWatcher {
	watchersLock.Lock()
	defer watchersLock.Unlock()
	k := util.GenerateKey(kubeconfigBytes, namespace)
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
	if load, ok := reverseInfo.Load(util.GenerateKey(n.kubeconfigBytes, n.namespace)); ok {
		for k, v := range load.(*name).resources.ele {
			if v.HasKey(resource) {
				return k
			}
		}
	}
	return ""
}

func (n *name) deleteByResource(resource string) {
	reverseInfoLock.Lock()
	defer reverseInfoLock.Unlock()
	if load, ok := reverseInfo.Load(util.GenerateKey(n.kubeconfigBytes, n.namespace)); ok {
		for _, v := range load.(*name).resources.ele {
			v.DeleteByKeys(resource)
		}
	}
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
	return util.GenerateKey(c.kubeconfigBytes, c.namespace)
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
	return util.GenerateKey(c.kubeconfigBytes, c.namespace) == util.GenerateKey(kubeconfigBytes, namespace)
}

func (c *ConnectInfo) IsSameCluster(kubeconfigBytes []byte) bool {
	return util.GenerateKey(kubeconfigBytes, "") == util.GenerateKey(c.kubeconfigBytes, "")
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
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	close(w.stopChan)
}

type resourceHandler struct {
	namespace       string
	kubeconfigBytes []byte
	reverseInfoLock *sync.Mutex
	reverseInfo     *sync.Map
}

func (h *resourceHandler) toKey() string {
	return util.GenerateKey(h.kubeconfigBytes, h.namespace)
}

func (h *resourceHandler) OnAdd(obj interface{}) {
	h.reverseInfoLock.Lock()
	defer h.reverseInfoLock.Unlock()

	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	if status.Connect.IsConnected() {
		if !connectInfo.IsEmpty() && !connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
			// todo someone already connected, release others or myself
			// release others
			release(connectInfo.kubeconfigBytes, connectInfo.namespace)
			notifySudoDaemonToDisConnect(connectInfo.kubeconfigBytes, connectInfo.namespace)
		}
		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) || connectInfo.IsEmpty() {
			go notifySudoDaemonToConnect(h.kubeconfigBytes, h.namespace)

			connectInfo.namespace = h.namespace
			connectInfo.kubeconfigBytes = h.kubeconfigBytes
			connectInfo.ip = status.MacToIP[util.GetMacAddress().String()]
		}
	}
	modifyReverseInfo(h, status)
}

func (h *resourceHandler) OnUpdate(oldObj, newObj interface{}) {
	h.reverseInfoLock.Lock()
	defer h.reverseInfoLock.Unlock()

	oldStatus := ToStatus(oldObj.(*corev1.ConfigMap).Data)
	newStatus := ToStatus(newObj.(*corev1.ConfigMap).Data)
	if newStatus.Connect.IsConnected() {
		if !connectInfo.IsEmpty() && !connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
			// todo someone already connected, release others or myself
			// release others
			release(connectInfo.kubeconfigBytes, connectInfo.namespace)
			notifySudoDaemonToDisConnect(connectInfo.kubeconfigBytes, connectInfo.namespace)
		}

		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) || connectInfo.IsEmpty() {
			go notifySudoDaemonToConnect(h.kubeconfigBytes, h.namespace)
			connectInfo.namespace = h.namespace
			connectInfo.kubeconfigBytes = h.kubeconfigBytes
			connectInfo.ip = newStatus.MacToIP[util.GetMacAddress().String()]
		}
	}
	// if connected --> disconnected, needs to notify sudo daemon to connect
	if oldStatus.Connect.IsConnected() && !newStatus.Connect.IsConnected() {
		if connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
			connectInfo.cleanup()
			go notifySudoDaemonToDisConnect(h.kubeconfigBytes, h.namespace)
		}
	}
	modifyReverseInfo(h, newStatus)
}

func (h *resourceHandler) OnDelete(obj interface{}) {
	h.reverseInfoLock.Lock()
	defer h.reverseInfoLock.Unlock()

	configMap := obj.(*corev1.ConfigMap)
	status := ToStatus(configMap.Data)
	h.reverseInfo.Delete(h.toKey())
	if status.Connect.IsConnected() && connectInfo.IsSame(h.kubeconfigBytes, h.namespace) {
		connectInfo.cleanup()
		notifySudoDaemonToDisConnect(h.kubeconfigBytes, h.namespace)
	}
}

// release resource handler will stop watcher
func release(kubeconfigBytes []byte, namespace string) {
	debug.PrintStack()
	// needs to notify sudo daemon to update connect namespace
	fmt.Println("needs to notify sudo daemon to update connect namespace, this should not to be happen")
	fmt.Printf("current: %s, to be releaded: %s\n", connectInfo.namespace, namespace)
	fmt.Printf("current: %s, to be releaded: %s\n", string(connectInfo.kubeconfigBytes), string(kubeconfigBytes))
	fmt.Printf("value: %s, value: %s, is same: %v\n", connectInfo.toKey(), util.GenerateKey(kubeconfigBytes, namespace),
		connectInfo.toKey() == util.GenerateKey(kubeconfigBytes, namespace))

	//ReleaseWatcher(kubeconfigBytes, namespace)
	if clientset, err := util.GetClientSetByKubeconfigBytes(kubeconfigBytes); err == nil {
		_ = UpdateConnect(clientset, namespace, func(list sets.String, item string) {
			list.Delete(item)
		})
		if value, found := GetReverseInfo().Load(util.GenerateKey(kubeconfigBytes, namespace)); found {
			for _, s := range value.(*name).resources.LoadAndDeleteBelongToMeResources().KeySet() {
				_ = UpdateReverseConfigMap(clientset, namespace, s,
					func(r *ReverseTotal, record ReverseRecord) {
						r.RemoveRecord(record)
					})
			}
		}
	}
	//GetOrGenerateConfigMapWatcher(kubeconfigBytes, namespace, nil)
}

// modifyReverseInfo modify reverse info using latest vpn status
// iterator latest vpn status and delete resource which not exist
func modifyReverseInfo(h *resourceHandler, latest VPNStatus) {
	// using same reference, because health check will modify status

	local, ok := h.reverseInfo.Load(h.toKey())
	if !ok {
		h.reverseInfo.Store(h.toKey(), &name{
			kubeconfigBytes: h.kubeconfigBytes,
			namespace:       h.namespace,
			resources:       latest.Reverse,
		})
		return
	}
	// add missing
	for macAddress, latestResources := range latest.Reverse.ele {
		if localResource, found := local.(*name).resources.ele[macAddress]; found {
			for _, resource := range latestResources.KeySet() {
				// if resource is not reversing, needs to delete it
				if !localResource.HasKey(resource) {
					localResource.InsertByKeys(resource)
				}
			}
		} else {
			// if latest status contains mac address that local cache not contains, needs to add it
			local.(*name).resources.ele[macAddress] = latestResources
		}
	}

	// remove useless
	for macAddress, localResources := range local.(*name).resources.ele {
		if latestResource, found := latest.Reverse.ele[macAddress]; !found {
			delete(local.(*name).resources.ele, macAddress)
		} else {
			for _, resource := range localResources.KeySet() {
				if !latestResource.HasKey(resource) {
					localResources.DeleteByKeys(resource)
				}
			}
		}
	}
}

func notifySudoDaemonToConnect(kubeconfigBytes []byte, namespace string) {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return
	}
	path := k8sutils.GetOrGenKubeConfigPath(string(kubeconfigBytes))
	readCloser, err := client.SendSudoVPNOperateCommand(path, namespace, command.Connect, "")
	if err == nil && readCloser != nil {
		_ = readCloser.Close()
	}
}

func notifySudoDaemonToDisConnect(kubeconfigBytes []byte, namespace string) {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return
	}
	path := k8sutils.GetOrGenKubeConfigPath(string(kubeconfigBytes))
	readCloser, err := client.SendSudoVPNOperateCommand(path, namespace, command.DisConnect, "")
	if err == nil && readCloser != nil {
		_ = readCloser.Close()
	}
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
		ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Second*5)
		defer cancelFunc()
		cmd := exec.CommandContext(ctx, "ping", "-c", "4", util.IpRange.String())
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
		if !connectInfo.IsSameCluster(value.(*name).kubeconfigBytes) {
			return true
		}
		path := k8sutils.GetOrGenKubeConfigPath(string(value.(*name).kubeconfigBytes))
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
			go func(connect *pkg.ConnectOptions, workload string, info *resourceInfo) {
				if _, err := connect.Shell(context.TODO(), workload); err != nil {
					info.Health = UnHealthy
				} else {
					info.Health = Healthy
				}
			}(connect, k, v)
		})
		return true
	})
}
