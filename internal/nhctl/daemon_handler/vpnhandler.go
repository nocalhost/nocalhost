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
	"nocalhost/pkg/nhctl/log"
	"strings"
	"time"
)

// HandleVPNOperate not sudo daemon, vpn controller
func HandleVPNOperate(cmd *command.VPNOperateCommand, writer io.WriteCloser) error {
	defer writer.Close()
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
		writer.Close()
		return err
	}
	if err := connect.Prepare(logCtx); err != nil {
		logger.Error(util.EndSignFailed)
		writer.Close()
		return err
	}
	_ = remote.NewDHCPManager(connect.GetClientSet(), cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary(logCtx)
	GetOrGenerateConfigMapWatcher(connect.KubeconfigBytes, cmd.Namespace, connect.GetClientSet().CoreV1().RESTClient())
	client, _ := daemon_client.GetDaemonClient(true)
	switch cmd.Action {
	case command.Connect:
		// change to another cluster or namespace, clean all reverse
		if !connectInfo.IsEmpty() && !connectInfo.IsSame(connect.KubeconfigBytes, cmd.Namespace) {
			clientset, err := util.GetClientSetByKubeconfigBytes(connectInfo.kubeconfigBytes)
			if err != nil {
				writer.Close()
				return err
			}
			// cleanup all reverse
			path := nocalhost.GetOrGenKubeConfigPath(string(connectInfo.kubeconfigBytes))
			for _, d := range GetReverseInfo().List() {
				_ = UpdateReverseConfigMap(clientset, connectInfo.namespace, d, func(r *ReverseTotal, record ReverseRecord) {
					r.RemoveRecord(record)
				})
				connectOptions := &pkg.ConnectOptions{
					Logger:         logger,
					KubeconfigPath: path,
					Namespace:      connectInfo.namespace,
					Workloads:      []string{d},
				}
				connectOptions.InitClient(context.TODO())
				if err = connectOptions.RemoveInboundPod(); err != nil {
					logger.Errorln(fmt.Sprintf(
						"delete reverse resource %s-%s error, err: %v", connectInfo.namespace, d, err))
				} else {
					logger.Infoln(fmt.Sprintf(
						"delete reverse resource %s-%s successfully", connectInfo.namespace, d))
				}
			}
			// disconnect from old cluster or namespace
			if err = UpdateConnect(clientset, connectInfo.namespace, func(list sets.String, item string) {
				list.Delete(item)
			}); err != nil {
				logger.Infof("error while remove connection info of %s\n", connectInfo.namespace)
			}
			if r, err := client.SendSudoVPNOperateCommand(path, connectInfo.namespace, command.DisConnect, cmd.Resource); err == nil {
				printStreamToOut(r)
			}
			// let informer notify daemon to take effect
			time.Sleep(time.Second * 2)
			connectInfo.Cleanup()
		}
		// connect to new cluster or namespace
		//if connectInfo.IsEmpty() {
		if err := UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
			list.Insert(address)
		}); err != nil {
			logger.Errorln(err)
			return err
		}
		if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource); err == nil {
			transStreamToWriter(writer, r)
		}
		//}

		// reverse resource if needed
		if len(cmd.Resource) != 0 {
			_ = UpdateReverseConfigMap(connect.GetClientSet(), cmd.Namespace, cmd.Resource, func(r *ReverseTotal, record ReverseRecord) {
				r.AddRecord(record)
			})
			if err := connect.DoReverse(context.TODO()); err != nil {
				logger.Errorln(err)
				logger.Infoln(util.EndSignFailed)
				return err
			}
		}
		logger.Infoln(util.EndSignOK)
		writer.Close()
		return nil
	case command.Reconnect:
		//err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		//err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
		return nil
	case command.DisConnect:
		if len(cmd.Resource) != 0 {
			_ = UpdateReverseConfigMap(connect.GetClientSet(), cmd.Namespace, cmd.Resource,
				func(r *ReverseTotal, record ReverseRecord) {
					r.RemoveRecord(record)
				},
			)
			if err := connect.RemoveInboundPod(); err != nil {
				logger.Errorf("error while delete reverse, info: %s-%s, err: %v",
					connect.Namespace, cmd.Resource, err)
				logger.Infoln(util.EndSignFailed)
				return err
			} else {
				logger.Infof("delete reverse, info: %s-%s secusefully",
					connect.Namespace, cmd.Resource)
				logger.Infoln(util.EndSignOK)
				return nil
			}
		} else {
			ReleaseWatcher(connect.KubeconfigBytes, cmd.Namespace)
			_ = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
				list.Delete(address)
			})
			for _, d := range GetReverseInfo().List() {
				_ = UpdateReverseConfigMap(connect.GetClientSet(), connectInfo.namespace, d, func(r *ReverseTotal, record ReverseRecord) {
					r.RemoveRecord(record)
				})
				if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, connectInfo.namespace, command.ReverseDisConnect, d); err == nil {
					transStreamToWriter(writer, r)
				}
			}
			if r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
				io.Copy(writer, r)
			}
			return nil
		}
	default:
		return nil
	}
}

func UpdateReverseConfigMap(clientSet *kubernetes.Clientset, namespace, resource string, f func(r *ReverseTotal, record ReverseRecord)) error {
	get, err := clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		log.Warn(err)
		return err
	}
	t := FromStringToReverseTotal(get.Data[util.REVERSE])
	f(t, NewReverseRecordWithWorkloads(resource))
	get.Data[util.REVERSE] = t.ToString()
	_, err = clientSet.CoreV1().ConfigMaps(resource).Update(context.TODO(), get, v1.UpdateOptions{})
	return err
}

func UpdateConnect(clientSet *kubernetes.Clientset, namespace string, f func(list sets.String, item string)) error {
	//if !connectInfo.IsEmpty() {
	//	// todo
	//	return nil
	//}
	get, err := clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		log.Warn(err)
		return err
	}
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

func printStreamToOut(r io.ReadCloser) {
	if r == nil {
		return
	}
	defer r.Close()
	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			return
		}
		fmt.Println(string(line))
		if strings.Contains(string(line), util.EndSignOK) || strings.Contains(string(line), util.EndSignFailed) {
			return
		}
	}
}

func transStreamToWriter(writer io.Writer, r io.ReadCloser) {
	if r == nil {
		return
	}
	defer r.Close()
	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			return
		}
		fmt.Println(string(line))
		writer.Write(line)
		writer.Write([]byte("\n"))
		if strings.Contains(string(line), util.EndSignOK) || strings.Contains(string(line), util.EndSignFailed) {
			return
		}
	}
}
