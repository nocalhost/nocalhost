package daemon_handler

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
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

type ReverseTotal struct {
	// mac address --> resources
	resources map[string]Set
}

func (t *ReverseTotal) ReversedResource() sets.String {
	result := sets.NewString()
	if t != nil && t.resources != nil {
		for _, v := range t.resources {
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

// HandleVPNStatus todo
func HandleVPNStatus() (interface{}, error) {
	return nil, nil
}

func FromStringToConnectInfo(str string) *ConnectTotal {
	result := &ConnectTotal{list: sets.NewString()}
	for _, s := range strings.Split(str, "\n") {
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

func ToStatus(m map[string]string) *status {
	return &status{
		reverse: FromStringToReverseTotal(m[util.REVERSE]),
		connect: FromStringToConnectInfo(m[util.Connect]),
		mac2ip:  remote.FromStringToMac2IP(m[util.MacToIP]),
		dhcp:    remote.FromStringToDHCP(m[util.DHCP]),
	}
}

func (t *ReverseTotal) AddRecord(r ReverseRecord) *ReverseTotal {
	if m, _ := t.resources[r.MacAddress]; m != nil {
		m.InsertByKeys(r.Resources.List()...)
	} else {
		t.resources[r.MacAddress] = NewSetByKeys(r.Resources.List()...)
	}
	return t
}

func (t *ReverseTotal) RemoveRecord(r ReverseRecord) *ReverseTotal {
	if m, _ := t.resources[r.MacAddress]; m != nil {
		m.DeleteByKeys(r.Resources.List()...)
		if m.Len() == 0 {
			delete(t.resources, r.MacAddress)
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
	t = &ReverseTotal{resources: map[string]Set{}}
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
	for k, v := range t.resources {
		sb.WriteString(fmt.Sprintf("%s%s%s\n", k, util.Splitter, strings.Join(v.KeySet(), ",")))
	}
	return sb.String()
}

func (t *ReverseTotal) GetBelongToMeResources() Set {
	if t.resources == nil {
		return NewSet()
	}
	s := t.resources[util.GetMacAddress().String()]
	if s != nil {
		return s
	}
	return NewSet()
}

func (t *ReverseTotal) LoadAndDeleteBelongToMeResources() Set {
	if t.resources == nil {
		return NewSet()
	}
	mac := util.GetMacAddress().String()
	defer delete(t.resources, mac)
	s := t.resources[mac]
	if s != nil {
		return s
	}
	return NewSet()
}
