package daemon_handler

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/typed/core/v1"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
	"sync"
)

var status *VPNStatus

type VPNStatus struct {
	Reverse ReverseTotal
	Connect ConnectTotal
}

type ReverseTotal struct {
	ele map[string]sets.String
}

type ConnectTotal struct {
	list sets.String
}

// map[string]context.CancelFunc
var maps sync.Map

// status
var statusMap sync.Map

func HandleVPNStatus(ns, kubeconfig, resource string, clientset kubernetes.Clientset) error {
	id := util.GenerateKey(ns, kubeconfig)
	if _, found := maps.Load(id); !found {
		maps.Store(id, true)
		go func() {
			defer maps.Delete(id)
			w, err := clientset.
				CoreV1().
				ConfigMaps(ns).
				Watch(context.TODO(),
					v12.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("metadata.name", util.TrafficManager).String(),
					},
				)
			if errors.IsNotFound(err) {
				return
			}
			for {
				select {
				case e := <-w.ResultChan():
					switch e.Type {
					case watch.Deleted:
						return
					case watch.Modified, watch.Added:
						statusMap.Store(id, ToStatus(e.Object.(*corev1.ConfigMap).Data))
					}
				}
			}
		}()
	}
	return nil
}

func FromStrToConnectTotal(string2 string) ConnectTotal {
	result := ConnectTotal{list: sets.NewString()}
	for _, s := range strings.Split(string2, "\n") {
		if len(s) != 0 {
			result.list.Insert(s)
		}
	}
	return result
}

func ToStatus(string2 map[string]string) VPNStatus {
	return VPNStatus{
		Reverse: FromStringToReverseTotal(string2[util.REVERSE]),
		Connect: FromStrToConnectTotal(string2[util.Connect]),
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

func FromStringToReverseTotal(s string) (t ReverseTotal) {
	t.ele = map[string]sets.String{}
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
