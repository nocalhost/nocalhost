package daemon_handler

import (
	"bufio"
	"context"
	"fmt"
	"io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
)

// HandleVPNOperate not sudo daemon, vpn controller
func HandleVPNOperate(cmd *command.VPNOperateCommand, writer *io.PipeWriter) error {
	logCtx := util.GetContextWithLogger(writer)
	logger := util.GetLoggerFromContext(logCtx)
	connect := &pkg.ConnectOptions{
		Logger:         logger,
		KubeconfigPath: cmd.KubeConfig,
		Namespace:      cmd.Namespace,
		Workloads:      []string{cmd.Resource},
	}
	if err := connect.InitClient(logCtx); err != nil {
		logger.Error(util.EndSignFailed)
		return err
	}
	if err := connect.Prepare(logCtx); err != nil {
		logger.Error(util.EndSignFailed)
		return err
	}
	_ = remote.NewDHCPManager(connect.GetClientSet(), cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary(logCtx)
	GetOrGenerateConfigMapWatcher(connect.KubeconfigBytes, cmd.Namespace, connect.GetClientSet().CoreV1().RESTClient())
	client, err := daemon_client.GetDaemonClient(true)
	switch cmd.Action {
	case command.Connect:
		if len(cmd.Resource) != 0 {
			if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Reverse, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
			err = Update(connect.GetClientSet(), cmd, func(r *ReverseTotal, record ReverseRecord) { r.AddRecord(record) })
		} else {
			// from one namespace to another
			if !connectInfo.IsEmpty() && !connectInfo.IsSame(connect.KubeconfigBytes, cmd.Namespace) {
				clientset, err := util.GetClientSetByKubeconfigBytes(connectInfo.kubeconfigBytes)
				if err != nil {
					return err
				}
				_ = UpdateConnect(clientset, connectInfo.namespace, func(list sets.String, address string) {
					list.Delete(address)
				})
				path := nocalhost.GetOrGenKubeConfigPath(string(connectInfo.kubeconfigBytes))
				if r, err := client.SendSudoVPNOperateCommand(path, connectInfo.namespace, command.DisConnect, ""); err == nil {
					reader := bufio.NewReader(r)
					for {
						line, _, err := reader.ReadLine()
						if err != nil {
							break
						}
						fmt.Println(string(line))
						if strings.Contains(string(line), util.EndSignOK) || strings.Contains(string(line), util.EndSignFailed) {
							r.Close()
							break
						}
					}
					//io.Copy(writer, r)
				}
			}
			_ = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
				list.Insert(address)
			})
			if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
		}
	case command.Reconnect:
		//err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		//err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
	case command.DisConnect:
		if len(cmd.Resource) != 0 {
			if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.ReverseDisConnect, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
		} else {
			if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
				if r != nil {
					go io.Copy(writer, r)
				}
			}
			UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
				list.Delete(address)
			})
			ReleaseWatcher(connect.KubeconfigBytes, cmd.Namespace)
		}
		err = Update(connect.GetClientSet(), cmd, func(r *ReverseTotal, record ReverseRecord) { r.RemoveRecord(record) })
	}
	return err
}

func Update(clientSet *kubernetes.Clientset, cmd *command.VPNOperateCommand, f func(r *ReverseTotal, record ReverseRecord)) error {
	get, err := clientSet.CoreV1().ConfigMaps(cmd.Namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		return err
	}
	t := FromStringToReverseTotal(get.Data[util.REVERSE])
	f(t, NewReverseRecordWithWorkloads(cmd.Resource))
	get.Data[util.REVERSE] = t.ToString()
	_, err = clientSet.CoreV1().ConfigMaps(cmd.Namespace).Update(context.TODO(), get, v1.UpdateOptions{})
	return err
}

func UpdateConnect(clientSet *kubernetes.Clientset, namespace string, f func(_ sets.String, _ string)) error {
	//if !connectInfo.IsEmpty() {
	//	// todo
	//	return nil
	//}
	get, err := clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	t := FromStrToConnectTotal(get.Data[util.Connect])
	f(t.list, util.GetMacAddress().String())
	get.Data[util.Connect] = t.ToString()
	_, err = clientSet.CoreV1().ConfigMaps(namespace).Update(context.TODO(), get, v1.UpdateOptions{})
	return err
}

// MacAddress1:deployment/test,service/test
// MacAddress2:deployment/hello,service/world

func init() {
	util.InitLogger(util.Debug)
}
