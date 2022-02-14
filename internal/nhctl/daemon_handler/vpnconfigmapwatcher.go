package daemon_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/k8sutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"sync"
	"time"
)

var watchers = map[string]*ConfigMapWatcher{}
var watchersLock = &sync.Mutex{}
var funcChan = make(chan func(), 1000)

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
	k := util.GenerateKey(kubeconfigBytes, namespace)
	watchersLock.Lock()
	defer watchersLock.Unlock()
	if v, ok := watchers[k]; ok && v != nil {
		return v
	}
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

var connectInfo = &ConnectInfo{}

var statusInfoLock = &sync.Mutex{}

// kubeconfig+ns --> name
var statusInfo = &sync.Map{}

type status struct {
	kubeconfigBytes []byte
	namespace       string
	// mac --> resources, this machine reverse those resources
	reverse *ReverseTotal
	// connection info, which machine connect to this namespace
	connect *ConnectTotal
	// mac to ip mapping
	mac2ip remote.DHCPRecordMap
	// mac --> ips, this mac address rent those ip, needs to release ips while disconnect
	dhcp map[string]sets.Int
}

func (n *status) getMacByResource(resource string) string {
	statusInfoLock.Lock()
	defer statusInfoLock.Unlock()
	if load, ok := statusInfo.Load(util.GenerateKey(n.kubeconfigBytes, n.namespace)); ok {
		for k, v := range load.(*status).reverse.resources {
			if v.HasKey(resource) {
				return k
			}
		}
	}
	return ""
}

func (n *status) deleteByResource(resource string) {
	statusInfoLock.Lock()
	defer statusInfoLock.Unlock()
	if load, ok := statusInfo.Load(util.GenerateKey(n.kubeconfigBytes, n.namespace)); ok {
		for _, v := range load.(*status).reverse.resources {
			v.DeleteByKeys(resource)
		}
	}
}

func (n *status) isReversingByMe(resource string) bool {
	if a := n.getMacByResource(resource); len(a) != 0 {
		if util.GetMacAddress().String() == a {
			return true
		}
	}
	return false
}

func (n *status) isReversingByOther(resource string) bool {
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
	return statusInfo
}

type ConnectInfo struct {
	uid             string
	kubeconfigBytes []byte
	namespace       string
	// kubeNs those who connect to this cluster
	kubeNs sets.String
	ip     string
	health HealthEnum
}

//func (c ConnectInfo) toKey() string {
//	return util.GenerateKey(c.kubeconfigBytes, c.namespace)
//}

func (c ConnectInfo) Status() string {
	return c.health.String()
}

func (c *ConnectInfo) cleanup() {
	c.ip = ""
	c.uid = ""
	c.kubeNs = sets.NewString()
	c.kubeconfigBytes = []byte{}
	c.namespace = ""
}

func (c *ConnectInfo) IsEmpty() bool {
	return c == nil || c.uid == ""
}

func (c *ConnectInfo) IsSameUid(uid string) bool {
	return c.uid == uid
}

func (c *ConnectInfo) IsSameCluster(kubeconfigBytes []byte) bool {
	return c.kubeNs.Has(util.GenerateKey(kubeconfigBytes, ""))
}

func (c *ConnectInfo) getIPIfIsMe(kubeconfigBytes []byte, namespace string) (ip string) {
	if c.kubeNs.Has(util.GenerateKey(kubeconfigBytes, namespace)) {
		ip = c.ip
	}
	return
}

func (c *ConnectInfo) GetUid() string {
	return c.uid
}

func (c *ConnectInfo) GetNamespace() string {
	return c.namespace
}

func (c *ConnectInfo) GetKubeconfig() string {
	return string(c.kubeconfigBytes)
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
		statusInfoLock:  statusInfoLock,
		statusInfo:      statusInfo,
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
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
	defer cancelFunc()
	cache.WaitForCacheSync(ctx.Done(), w.informer.HasSynced)
}

func (w *ConfigMapWatcher) Stop() {
	defer func() {
		if r := recover(); r != nil {
		}
	}()
	close(w.stopChan)
}

type resourceHandler struct {
	// unique id
	uid             string
	namespace       string
	kubeconfigBytes []byte
	statusInfoLock  *sync.Mutex
	statusInfo      *sync.Map
}

func (h *resourceHandler) toKey() string {
	return util.GenerateKey(h.kubeconfigBytes, h.namespace)
}

func (h *resourceHandler) OnAdd(obj interface{}) {
	h.statusInfoLock.Lock()
	defer h.statusInfoLock.Unlock()

	configMap := obj.(*corev1.ConfigMap)
	h.uid = string(configMap.GetUID())
	toStatus := ToStatus(configMap.Data)
	modifyReverseInfo(h, toStatus)
	backup := *connectInfo
	// if connect to a cluster, needs to keep it connect
	if toStatus.connect.IsConnected() {
		connectInfo.uid = h.uid
		connectInfo.namespace = h.namespace
		connectInfo.kubeconfigBytes = h.kubeconfigBytes
		connectInfo.ip = toStatus.mac2ip.GetIPByMac(util.GetMacAddress().String())
		connectInfo.kubeNs = sets.NewString(
			util.GenerateKey(h.kubeconfigBytes, h.namespace), util.GenerateKey(h.kubeconfigBytes, ""),
		)
		// if is connected to other cluster, needs to disconnect it
		funcChan <- func() {
			if !backup.IsEmpty() && !backup.IsSameUid(h.uid) {
				// release others
				release(backup.kubeconfigBytes, backup.namespace)
			}
			// connect to this cluster
			notifySudoDaemonToConnect(h.uid, h.kubeconfigBytes, h.namespace)
		}
	}
}

func (h *resourceHandler) OnUpdate(oldObj, newObj interface{}) {
	h.statusInfoLock.Lock()
	defer h.statusInfoLock.Unlock()

	oldStatus := ToStatus(oldObj.(*corev1.ConfigMap).Data)
	newStatus := ToStatus(newObj.(*corev1.ConfigMap).Data)
	modifyReverseInfo(h, newStatus)
	backup := *connectInfo
	// if connect to a cluster, needs to keep it connect
	if newStatus.connect.IsConnected() {
		connectInfo.uid = h.uid
		connectInfo.namespace = h.namespace
		connectInfo.kubeconfigBytes = h.kubeconfigBytes
		connectInfo.ip = newStatus.mac2ip.GetIPByMac(util.GetMacAddress().String())
		connectInfo.kubeNs = sets.NewString(
			util.GenerateKey(h.kubeconfigBytes, h.namespace), util.GenerateKey(h.kubeconfigBytes, ""),
		)
		funcChan <- func() {
			// if is connected to other cluster, needs to disconnect it
			if !backup.IsEmpty() && !backup.IsSameUid(h.uid) {
				// release others
				release(backup.kubeconfigBytes, backup.namespace)
			}
			// connect to this cluster
			notifySudoDaemonToConnect(h.uid, h.kubeconfigBytes, h.namespace)
		}
	} else
	// if connected --> disconnected, needs to notify sudo daemon to disconnect
	// other user can close vpn you create
	if oldStatus.connect.IsConnected() && !newStatus.connect.IsConnected() {
		if backup.IsSameUid(h.uid) {
			connectInfo.cleanup()
			funcChan <- func() {
				notifySudoDaemonToDisConnect(h.uid, h.kubeconfigBytes, h.namespace)
			}
		}
	}

}

// OnDelete will not release watcher, keep watching
func (h *resourceHandler) OnDelete(obj interface{}) {
	h.statusInfoLock.Lock()
	defer h.statusInfoLock.Unlock()

	configMap := obj.(*corev1.ConfigMap)
	toStatus := ToStatus(configMap.Data)
	h.statusInfo.Delete(h.toKey())
	// if this machine is connected, needs to disconnect vpn, but still keep watching configmap
	if toStatus.connect.IsConnected() && connectInfo.IsSameUid(h.uid) {
		connectInfo.cleanup()
		funcChan <- func() {
			notifySudoDaemonToDisConnect(h.uid, h.kubeconfigBytes, h.namespace)
		}
	}
}

// release resource handler will stop watcher
func release(kubeconfigBytes []byte, namespace string) {
	path := k8sutils.GetOrGenKubeConfigPath(string(kubeconfigBytes))
	if err := disconnectedFromNamespace(context.TODO(), os.Stdout, path, namespace); err != nil {
		log.Warn(err)
	}
}

// modifyReverseInfo modify reverse info using latest vpn status
// iterator latest vpn status and delete resource which not exist
func modifyReverseInfo(h *resourceHandler, latest *status) {
	// using same reference, because health check will modify status
	local, ok := h.statusInfo.Load(h.toKey())
	if !ok {
		h.statusInfo.Store(h.toKey(), &status{
			kubeconfigBytes: h.kubeconfigBytes,
			namespace:       h.namespace,
			reverse:         latest.reverse,
		})
		return
	}
	// add missing
	for macAddress, latestResources := range latest.reverse.resources {
		if localResource, found := local.(*status).reverse.resources[macAddress]; found {
			for _, resource := range latestResources.KeySet() {
				// if resource is not reversing, needs to delete it
				if !localResource.HasKey(resource) {
					localResource.InsertByKeys(resource)
				}
			}
		} else {
			// if latest status contains mac address that local cache not contains, needs to add it
			local.(*status).reverse.resources[macAddress] = latestResources
		}
	}

	// remove useless
	for macAddress, localResources := range local.(*status).reverse.resources {
		if latestResource, found := latest.reverse.resources[macAddress]; !found {
			delete(local.(*status).reverse.resources, macAddress)
		} else {
			for _, resource := range localResources.KeySet() {
				if !latestResource.HasKey(resource) {
					localResources.DeleteByKeys(resource)
				}
			}
		}
	}
}

func notifySudoDaemonToConnect(uid string, kubeconfigBytes []byte, namespace string) {
	if !util.IsPortListening(daemon_common.SudoDaemonPort) {
		return
	}
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return
	}
	if info, err := getSudoConnectInfo(); err == nil && !info.IsEmpty() {
		if info.IsSameUid(uid) {
			return
		}
		// disconnect from current cluster
		path := k8sutils.GetOrGenKubeConfigPath(string(info.kubeconfigBytes))
		if err = client.SendSudoVPNOperateCommand(path, info.namespace, command.DisConnect, func(r io.Reader) error {
			if ok := transStreamToWriter(r, os.Stdout); !ok {
				log.Warnf("can not disconnect from kubeconfig: %s", path)
				return fmt.Errorf("can not disconnect from kubeconfig: %s", path)
			}
			return nil
		}); err != nil {
			return
		}
		time.Sleep(time.Second * 1)
	}
	path := k8sutils.GetOrGenKubeConfigPath(string(kubeconfigBytes))
	_ = client.SendSudoVPNOperateCommand(path, namespace, command.Connect, func(reader io.Reader) error {
		if ok := transStreamToWriter(reader, os.Stdout); !ok {
			log.Warnf("can not connect to kubeconfig: %s", path)
		}
		return nil
	})
}

// disconnect from special cluster
func notifySudoDaemonToDisConnect(uid string, kubeconfigBytes []byte, namespace string) {
	if !util.IsPortListening(daemon_common.SudoDaemonPort) {
		return
	}
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return
	}

	if info, err := getSudoConnectInfo(); err == nil {
		// if sudo daemon is connecting to cluster which want's to connect, do nothing
		if info.IsEmpty() {
			return
		}
		// if sudo daemon is not connect to this cluster, no needs to disconnect from it
		if !info.IsSameUid(uid) {
			return
		}
	}
	path := k8sutils.GetOrGenKubeConfigPath(string(kubeconfigBytes))
	_ = client.SendSudoVPNOperateCommand(path, namespace, command.DisConnect, func(reader io.Reader) error {
		_, err = io.Copy(os.Stdout, reader)
		return err
	})
}

func getSudoConnectInfo() (info *ConnectInfo, err error) {
	if client, err := daemon_client.GetDaemonClient(true); err == nil {
		if obj, err := client.SendSudoVPNStatusCommand(); err == nil {
			if bytes, err := json.Marshal(obj); err == nil {
				var result pkg.ConnectOptions
				if err = json.Unmarshal(bytes, &result); err == nil {
					return &ConnectInfo{
						uid:             result.Uid,
						kubeconfigBytes: result.KubeconfigBytes,
						namespace:       result.Namespace,
					}, nil
				}
			}
		}
	}
	return
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
						if err := recover(); err != nil {
						}
					}()
					go checkConnect()
					go checkReverse()
					go communicateEachOther()
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
		if cmd.ProcessState != nil && cmd.ProcessState.Success() {
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
		if !connectInfo.IsSameCluster(value.(*status).kubeconfigBytes) {
			return true
		}
		path := k8sutils.GetOrGenKubeConfigPath(string(value.(*status).kubeconfigBytes))
		connect := &pkg.ConnectOptions{
			KubeconfigPath: path,
			Namespace:      value.(*status).namespace,
			Workloads:      []string{},
		}
		if err := connect.InitClient(context.TODO()); err != nil {
			return true
		}
		if err := connect.Prepare(context.TODO()); err != nil {
			return true
		}
		value.(*status).reverse.GetBelongToMeResources().ForEach(func(k string, v *resourceInfo) {
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

func communicateEachOther() {
	if connectInfo.IsEmpty() {
		return
	}

	if w := GetOrGenerateConfigMapWatcher(connectInfo.kubeconfigBytes, connectInfo.namespace, nil); w != nil {
		for _, i := range w.informer.GetStore().List() {
			if cm, ok := i.(*corev1.ConfigMap); ok {
				dhcp := remote.FromStringToDHCP(cm.Data[util.DHCP])
				if v, found := dhcp[util.GetMacAddress().String()]; found {
					for _, ip := range v.List() {
						_, _ = util.Ping(fmt.Sprintf("223.254.254.%v", ip))
					}
				}
			}
		}
	}
}

func init() {
	go func() {
		for f := range funcChan {
			f()
		}
	}()
}
