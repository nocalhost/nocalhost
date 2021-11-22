package daemon_handler

import (
	"context"
	"fmt"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
)

// HandleVPNOperate not sudo daemon, vpn controller
func HandleVPNOperate(cmd *command.VPNOperateCommand) error {
	file, _ := ioutil.ReadFile(cmd.KubeConfig)
	c, err := clientcmd.RESTConfigFromKubeConfig(file)
	if err != nil {
		return err
	}
	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}
	_ = remote.NewDHCPManager(clientSet, cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary()
	client, err := daemon_client.GetDaemonClient(true)
	switch cmd.Action {
	case command.Connect:
		if len(cmd.Resource) != 0 {
			err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Reverse, cmd.Resource)
		} else {
			err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
		}
		err = Update(clientSet, cmd, func(r *ReverseTotal, record ReverseRecord) { r.AddRecord(record) })
	case command.Reconnect:
		err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
		err = Update(clientSet, cmd, func(r *ReverseTotal, record ReverseRecord) { r.AddRecord(record) })
	case command.DisConnect:
		if len(cmd.Resource) != 0 {
			err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.ReverseDisConnect, cmd.Resource)
		} else {
			err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		}
		err = Update(clientSet, cmd, func(r *ReverseTotal, record ReverseRecord) { r.RemoveRecord(record) })
	}
	return err
}

func Update(clientSet *kubernetes.Clientset, cmd *command.VPNOperateCommand, f func(r *ReverseTotal, record ReverseRecord)) error {
	get, err := clientSet.CoreV1().ConfigMaps(cmd.Namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		return err
	}
	t := FromStringToReverseTotal(get.Data[util.REVERSE])
	f(&t, NewReverseRecordWithWorkloads(cmd.Resource))
	get.Data[util.REVERSE] = t.ToString()
	_, err = clientSet.CoreV1().ConfigMaps(cmd.Namespace).Update(context.TODO(), get, v1.UpdateOptions{})
	return err
}

type ReverseTotal struct {
	ele map[string]sets.String
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

func (t *ReverseTotal) GetBelongToMeResources() sets.String {
	s := t.ele[util.GetMacAddress().String()]
	if s != nil {
		return s
	}
	return sets.NewString()
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

// MacAddress1:deployment/test,service/test
// MacAddress2:deployment/hello,service/world

func FromStringToReverseTotal(s string) (t ReverseTotal) {
	t.ele = map[string]sets.String{}
	if len(s) == 0 {
		return
	}
	itemList := strings.Split(s, "\n")
	for _, item := range itemList {
		if strings.Count(item, ":") == 1 {
			i := strings.Split(item, ":")
			t.AddRecord(ReverseRecord{MacAddress: i[0], Resources: sets.NewString(strings.Split(i[1], ",")...)})
		}
	}
	return
}

func (t *ReverseTotal) ToString() string {
	var sb strings.Builder
	for k, v := range t.ele {
		sb.WriteString(fmt.Sprintf("%s:%s\n", k, strings.Join(v.List(), ",")))
	}
	return sb.String()
}

func IsBelongToMe(configMapInterface coreV1.ConfigMapInterface, resources string) (bool, error) {
	get, err := configMapInterface.Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
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
