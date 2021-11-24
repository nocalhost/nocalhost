package daemon_handler

import (
	"context"
	"io"
	"io/ioutil"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
)

// HandleVPNOperate not sudo daemon, vpn controller
func HandleVPNOperate(cmd *command.VPNOperateCommand, writer *io.PipeWriter) error {
	ctx := util.GetContextWithLogger(writer)
	file, _ := ioutil.ReadFile(cmd.KubeConfig)
	c, err := clientcmd.RESTConfigFromKubeConfig(file)
	if err != nil {
		return err
	}
	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		return err
	}
	_ = remote.NewDHCPManager(clientSet, cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary(ctx)
	client, err := daemon_client.GetDaemonClient(true)
	switch cmd.Action {
	case command.Connect:
		if len(cmd.Resource) != 0 {
			if r, err := client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Reverse, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
			err = Update(clientSet, cmd, func(r *ReverseTotal, record ReverseRecord) { r.AddRecord(record) })
		} else {
			if r, err := client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
		}
	case command.Reconnect:
		//err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		//err = client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
	case command.DisConnect:
		if len(cmd.Resource) != 0 {
			if r, err := client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.ReverseDisConnect, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
		} else {
			if r, err := client.SendToSudoDaemonVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
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

// MacAddress1:deployment/test,service/test
// MacAddress2:deployment/hello,service/world

func init() {
	util.InitLogger(util.Debug)
}
