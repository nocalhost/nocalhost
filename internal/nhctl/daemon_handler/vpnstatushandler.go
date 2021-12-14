package daemon_handler

import (
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
	"sync"
)

type HealthEnum string
type ModeEnum string

const (
	Unknown     HealthEnum = "unknown"
	UnHealthy   HealthEnum = "unHealthy"
	Healthy     HealthEnum = "healthy"
	ReverseMode ModeEnum   = "reverse"
	ConnectMode ModeEnum   = "connect"
)

func (e HealthEnum) String() string {
	return string(e)
}

func (m ModeEnum) String() string {
	return string(m)
}

//var status *VPNStatus
//var statusLock sync.Mutex

// status of mine
var connectHealthStatus *Func

// resources --> health
var reverseHeathStatus sync.Map

type Func struct {
	health HealthEnum
	f      context.CancelFunc
}

type VPNStatus struct {
	Reverse *ReverseTotal
	Connect *ConnectTotal
	MacToIP map[string]string
}

type ReverseTotal struct {
	// mac address --> resources
	ele map[string]Set
}

func (t *ReverseTotal) ReversedResource() sets.String {
	result := sets.NewString()
	if t != nil && t.ele != nil {
		for _, v := range t.ele {
			result.Insert(v.KeySet()...)
		}
	}
	return result
}

type ConnectTotal struct {
	// mac address list
	list sets.String
}

func (c *ConnectTotal) IsConnected() bool {
	return c.list.Has(util.GetMacAddress().String())
}

// todo
func HandleVPNStatus(cmd *command.VPNOperateCommand) (interface{}, error) {
	kubeconfigBytes, _ := ioutil.ReadFile(cmd.KubeConfig)
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, err
	}
	clientset, err1 := kubernetes.NewForConfig(config)
	if err1 != nil {
	}
	GetOrGenerateConfigMapWatcher(kubeconfigBytes, cmd.Namespace, clientset.CoreV1().RESTClient())
	if connectInfo.IsEmpty() {
	}
	return nil, nil
}

func FromStrToConnectTotal(string2 string) *ConnectTotal {
	result := &ConnectTotal{list: sets.NewString()}
	for _, s := range strings.Split(string2, "\n") {
		if len(s) != 0 {
			result.list.Insert(s)
		}
	}
	return result
}

func (c *ConnectTotal) ToString() string {
	var sb strings.Builder
	for _, s := range c.list.List() {
		sb.WriteString(s + "\n")
	}
	return sb.String()
}

func ToStatus(m map[string]string) VPNStatus {
	return VPNStatus{
		Reverse: FromStringToReverseTotal(m[util.REVERSE]),
		Connect: FromStrToConnectTotal(m[util.Connect]),
		MacToIP: remote.ToDHCP(m[util.MacToIP]).MacToIP(),
	}
}

func (t *ReverseTotal) AddRecord(r ReverseRecord) *ReverseTotal {
	if m, _ := t.ele[r.MacAddress]; m != nil {
		m.InsertByKeys(r.Resources.List()...)
	} else {
		t.ele[r.MacAddress] = NewSetByKeys(r.Resources.List()...)
	}
	return t
}

func (t *ReverseTotal) RemoveRecord(r ReverseRecord) *ReverseTotal {
	if m, _ := t.ele[r.MacAddress]; m != nil {
		m.DeleteByKeys(r.Resources.List()...)
		if m.Len() == 0 {
			delete(t.ele, r.MacAddress)
		}
	}
	return t
}

type ReverseRecord struct {
	MacAddress string
	Resources  sets.String
}

func NewReverseRecord(resourceType, resourceName string) ReverseRecord {
	return ReverseRecord{
		MacAddress: util.GetMacAddress().String(),
		Resources:  sets.NewString(fmt.Sprintf("%s/%s", resourceType, resourceName)),
	}
}

func NewReverseRecordWithWorkloads(workloads string) ReverseRecord {
	return ReverseRecord{
		MacAddress: util.GetMacAddress().String(),
		Resources:  sets.NewString(workloads),
	}
}

func FromStringToReverseTotal(s string) (t *ReverseTotal) {
	t = &ReverseTotal{ele: map[string]Set{}}
	if len(s) == 0 {
		return
	}
	itemList := strings.Split(s, "\n")
	for _, item := range itemList {
		if strings.Count(item, util.Splitter) == 1 {
			i := strings.Split(item, util.Splitter)
			t.AddRecord(ReverseRecord{MacAddress: i[0], Resources: sets.NewString(strings.Split(i[1], ",")...)})
		}
	}
	return
}

func (t *ReverseTotal) ToString() string {
	var sb strings.Builder
	for k, v := range t.ele {
		sb.WriteString(fmt.Sprintf("%s%s%s\n", k, util.Splitter, strings.Join(v.KeySet(), ",")))
	}
	return sb.String()
}

func (t *ReverseTotal) GetBelongToMeResources() Set {
	if t.ele == nil {
		return NewSet()
	}
	s := t.ele[util.GetMacAddress().String()]
	if s != nil {
		return s
	}
	return NewSet()
}

func (t *ReverseTotal) LoadAndDeleteBelongToMeResources() Set {
	if t.ele == nil {
		return NewSet()
	}
	mac := util.GetMacAddress().String()
	defer delete(t.ele, mac)
	s := t.ele[mac]
	if s != nil {
		return s
	}
	return NewSet()
}
