package daemon_handler

import (
	"context"
	"fmt"
	"io/ioutil"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
	"sync"
)

type Health string

const (
	DisConnected     Health = "DisConnected"
	ConnectHealth    Health = "ConnectHealth"
	ConnectUnHealth  Health = "ConnectUnHealth"
	NotReversed      Health = "NotReversed"
	ReversedHealth   Health = "ReversedHealth"
	ReversedUnHealth Health = "ReversedUnHealth"
)

//var status *VPNStatus
//var statusLock sync.Mutex

// status of mine
var connectHealthStatus *Func

// resources --> health
var reverseHeathStatus sync.Map

type Func struct {
	health Health
	f      context.CancelFunc
}

type VPNStatus struct {
	Reverse *ReverseTotal
	Connect *ConnectTotal
}

type ReverseTotal struct {
	// mac address --> resources
	ele map[string]sets.String
}

type ConnectTotal struct {
	// mac address list
	list sets.String
}

func (c *ConnectTotal) IsConnected() bool {
	return c.list.Has(util.GetMacAddress().String())
}

var defaultStatus = HealthStatus{ConnectStatus: DisConnected, ReserveStatus: NotReversed}

func HandleVPNStatus(cmd *command.VPNOperateCommand) (HealthStatus, error) {
	kubeconfigBytes, _ := ioutil.ReadFile(cmd.KubeConfig)
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return defaultStatus, err
	}
	clientset, err1 := kubernetes.NewForConfig(config)
	if err1 != nil {
		return defaultStatus, err1
	}
	GetOrGenerateConfigMapWatcher(kubeconfigBytes, cmd.Namespace, clientset.CoreV1().RESTClient())
	if connectInfo.IsEmpty() {
		return defaultStatus, nil
	}
	var reverseStatus, connectStatus Health
	if Reverse(kubeconfigBytes, cmd.Namespace, cmd.Resource) {
		if v, found := reverseHeathStatus.Load(cmd.Resource); found && v != nil {
			f := v.(*Func)
			reverseStatus = f.health
		} else {
			reverseStatus = ReversedUnHealth
		}
	} else {
		reverseStatus = NotReversed
	}
	if !connectInfo.IsEmpty() &&
		(string(connectInfo.kubeconfigBytes) == string(kubeconfigBytes)) &&
		connectInfo.namespace == cmd.Namespace {
		if connectHealthStatus != nil {
			connectStatus = connectHealthStatus.health
		} else {
			connectStatus = ConnectUnHealth
		}
	} else {
		connectStatus = DisConnected
	}

	return HealthStatus{ConnectStatus: connectStatus, ReserveStatus: reverseStatus}, nil
}

type HealthStatus struct {
	ConnectStatus Health `json:"connectStatus" yaml:"connectStatus"`
	ReserveStatus Health `json:"reserveStatus" yaml:"reserveStatus"`
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
	}
}

func (t *ReverseTotal) AddRecord(r ReverseRecord) *ReverseTotal {
	if m, _ := t.ele[r.MacAddress]; m != nil {
		m.Insert(r.Resources.List()...)
	} else {
		t.ele[r.MacAddress] = sets.NewString(r.Resources.List()...)
	}
	return t
}

func (t *ReverseTotal) RemoveRecord(r ReverseRecord) *ReverseTotal {
	if m, _ := t.ele[r.MacAddress]; m != nil {
		m.Delete(r.Resources.List()...)
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
	t = &ReverseTotal{ele: map[string]sets.String{}}
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
		sb.WriteString(fmt.Sprintf("%s%s%s\n", k, util.Splitter, strings.Join(v.List(), ",")))
	}
	return sb.String()
}

func (t *ReverseTotal) GetBelongToMeResources() sets.String {
	s := t.ele[util.GetMacAddress().String()]
	if s != nil {
		return s
	}
	return sets.NewString()
}

func IsBelongToMe(configMapInterface v1.ConfigMapInterface, resources string) (bool, error) {
	get, err := configMapInterface.Get(context.TODO(), util.TrafficManager, v12.GetOptions{})
	if err != nil {
		return false, err
	}
	s := get.Data[util.REVERSE]
	if len(s) == 0 {
		return false, nil
	}
	t := FromStringToReverseTotal(get.Data[util.REVERSE])
	return t.GetBelongToMeResources().Has(resources), nil
}
