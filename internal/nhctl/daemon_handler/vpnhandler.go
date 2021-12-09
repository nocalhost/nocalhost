package daemon_handler

import (
	"bufio"
	"context"
	"errors"
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
func HandleVPNOperate(cmd *command.VPNOperateCommand, writer io.WriteCloser) (err error) {
	preCheck(cmd)
	logCtx := util.GetContextWithLogger(writer)
	logger := util.GetLoggerFromContext(logCtx)
	defer func() {
		if err != nil {
			logger.Errorln(err)
			logger.Errorln(util.EndSignFailed)
		} else {
			logger.Infoln(util.EndSignOK)
		}
		writer.Close()
	}()
	connect := &pkg.ConnectOptions{
		Logger:         logger,
		KubeconfigPath: cmd.KubeConfig,
		Namespace:      cmd.Namespace,
		Workloads:      []string{cmd.Resource},
	}
	if err = connect.InitClient(logCtx); err != nil {
		return
	}
	if err = connect.Prepare(logCtx); err != nil {
		return
	}
	_ = remote.NewDHCPManager(connect.GetClientSet(), cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary(logCtx)
	w := GetOrGenerateConfigMapWatcher(connect.KubeconfigBytes, cmd.Namespace, connect.GetClientSet().CoreV1().RESTClient())
	if w == nil {
		logger.Printf("create configmap for ns: %s failed\n", cmd.Namespace)
	} else {
		logger.Printf("create configmap for ns: %s ok\n", cmd.Namespace)
	}
	client, _ := daemon_client.GetDaemonClient(true)
	switch cmd.Action {
	case command.Connect:
		// pre-check if resource already in reversing mode
		if load, ok := GetReverseInfo().Load(generateKey(connect.KubeconfigBytes, connect.Namespace)); ok {
			if load.(*name).resources.ReversedResource().Has(cmd.Resource) {
				err = errors.New("already in reversing mode")
				return
			}
		}

		// change to another cluster or namespace, clean all reverse
		if !connectInfo.IsEmpty() && !connectInfo.IsSame(connect.KubeconfigBytes, cmd.Namespace) {
			clientset, err := util.GetClientSetByKubeconfigBytes(connectInfo.kubeconfigBytes)
			if err != nil {
				return err
			}
			// cleanup all reverse
			path := nocalhost.GetOrGenKubeConfigPath(string(connectInfo.kubeconfigBytes))
			GetReverseInfo().Range(func(key, value interface{}) bool {
				if value == nil {
					return true
				}
				connectOptions := &pkg.ConnectOptions{
					Logger:         logger,
					KubeconfigPath: path,
					Namespace:      connectInfo.namespace,
				}
				if err = connectOptions.InitClient(context.TODO()); err != nil {
					return true
				}
				for _, d := range value.(*name).resources.GetBelongToMeResources().List() {
					_ = UpdateReverseConfigMap(clientset, connectInfo.namespace, d, func(r *ReverseTotal, record ReverseRecord) {
						r.RemoveRecord(record)
					})
					connectOptions.Workloads = []string{d}
					if err = connectOptions.RemoveInboundPod(); err != nil {
						logger.Errorln(fmt.Sprintf(
							"delete reverse resource %s-%s error, err: %v", connectInfo.namespace, d, err))
					} else {
						logger.Infoln(fmt.Sprintf(
							"delete reverse resource %s-%s successfully", connectInfo.namespace, d))
					}
				}
				return true
			})
			// disconnect from old cluster or namespace
			if err = UpdateConnect(clientset, connectInfo.namespace, func(list sets.String, item string) {
				list.Delete(item)
			}); err != nil {
				logger.Infof("error while remove connection info of %s\n", connectInfo.namespace)
			}
			if r, err := client.SendSudoVPNOperateCommand(
				path, connectInfo.namespace, command.DisConnect, cmd.Resource); err == nil {
				transStreamToWriterWithoutExit(writer, r)
			}
			// let informer notify daemon to take effect
			time.Sleep(time.Second * 2)
			connectInfo.cleanup()
		}
		// connect to new cluster or namespace
		//if connectInfo.IsEmpty() {
		if err = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
			list.Insert(address)
		}); err != nil {
			return
		}
		if r, err := client.SendSudoVPNOperateCommand(
			cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource); err == nil {
			transStreamToWriterWithoutExit(writer, r)
		}
		//}

		// reverse resource if needed
		if len(cmd.Resource) != 0 {
			_ = UpdateReverseConfigMap(connect.GetClientSet(), cmd.Namespace, cmd.Resource, func(r *ReverseTotal, record ReverseRecord) {
				r.AddRecord(record)
			})
			if err = connect.DoReverse(context.TODO()); err != nil {
				return
			}
		}
		return
	case command.Reconnect:
		if len(cmd.Resource) != 0 {
			return connect.DoReverse(context.TODO())
		}
		//err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		r, err := client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
		if err != nil {
			return err
		} else {
			transStreamToWriter(writer, r)
			return nil
		}
	case command.DisConnect:
		defer writer.Close()
		if len(cmd.Resource) != 0 {
			load, ok := GetReverseInfo().Load(generateKey(connect.KubeconfigBytes, connect.Namespace))
			if !ok {
				return nil
			}
			address := load.(*name).getMacByResource(cmd.Resource)
			_ = UpdateReverseConfigMap(connect.GetClientSet(), cmd.Namespace, cmd.Resource,
				func(r *ReverseTotal, record ReverseRecord) {
					record.MacAddress = address
					r.RemoveRecord(record)
				},
			)
			if err = connect.RemoveInboundPod(); err != nil {
				logger.Errorf("error while delete reverse, info: %s-%s, err: %v",
					connect.Namespace, cmd.Resource, err)
			} else {
				logger.Infof("delete reverse, info: %s-%s secusefully",
					connect.Namespace, cmd.Resource)
			}

			// if cancel last reverse resources, needs to close connect
			if value, found := GetReverseInfo().Load(generateKey(connect.KubeconfigBytes, connect.Namespace)); found {
				if value.(*name).resources.GetBelongToMeResources().Len() == 0 {
					ReleaseWatcher(connect.KubeconfigBytes, cmd.Namespace)
					_ = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
						list.Delete(address)
					})
					if r, err := client.SendSudoVPNOperateCommand(
						cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
						transStreamToWriter(writer, r)
					}
				}
			}
			return
		} else {
			ReleaseWatcher(connect.KubeconfigBytes, cmd.Namespace)
			_ = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
				list.Delete(address)
			})
			GetReverseInfo().Range(func(key, value interface{}) bool {
				if value == nil {
					return true
				}
				clientset, err := util.GetClientSetByKubeconfigBytes(value.(*name).kubeconfigBytes)
				if err != nil {
					return true
				}
				path := nocalhost.GetOrGenKubeConfigPath(string(value.(*name).kubeconfigBytes))
				temp := &pkg.ConnectOptions{
					Logger:         logger,
					KubeconfigPath: path,
					Namespace:      value.(*name).namespace,
					Workloads:      []string{},
				}
				if err = temp.InitClient(context.TODO()); err != nil {
					return true
				}
				for _, resource := range value.(*name).resources.GetBelongToMeResources().List() {
					_ = UpdateReverseConfigMap(clientset,
						value.(*name).namespace,
						resource,
						func(r *ReverseTotal, record ReverseRecord) {
							r.RemoveRecord(record)
						})
					temp.Workloads = []string{resource}
					_ = temp.RemoveInboundPod()
				}
				return true
			})
			if r, err := client.SendSudoVPNOperateCommand(
				cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
				transStreamToWriter(writer, r)
			}
			return
		}
	default:
		return errors.New("Unsupported operation: %s" + string(cmd.Action))
	}
}

func UpdateReverseConfigMap(
	clientSet *kubernetes.Clientset,
	namespace,
	resource string,
	f func(r *ReverseTotal, record ReverseRecord),
) error {
	get, err := clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		log.Warn(err)
		return err
	}
	t := FromStringToReverseTotal(get.Data[util.REVERSE])
	f(t, NewReverseRecordWithWorkloads(resource))
	get.Data[util.REVERSE] = t.ToString()
	_, err = clientSet.CoreV1().ConfigMaps(namespace).Update(context.TODO(), get, v1.UpdateOptions{})
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
		writer.Write(line)
		writer.Write([]byte("\n"))
		if strings.Contains(string(line), util.EndSignOK) || strings.Contains(string(line), util.EndSignFailed) {
			return
		}
	}
}

func transStreamToWriterWithoutExit(writer io.Writer, r io.ReadCloser) {
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
		if strings.Contains(string(line), util.EndSignOK) || strings.Contains(string(line), util.EndSignFailed) {
			return
		}
		writer.Write(line)
		writer.Write([]byte("\n"))
	}
}

func preCheck(cmd *command.VPNOperateCommand) {
	if len(cmd.Resource) != 0 {
		tuple, b, err := util.SplitResourceTypeName(cmd.Resource)
		if err == nil && b {
			switch strings.ToLower(tuple.Resource) {
			case "deployment", "deployments":
				tuple.Resource = "deployments"
			case "statefulset", "statefulsets":
				tuple.Resource = "statefulsets"
			case "replicaset", "replicasets":
				tuple.Resource = "replicasets"
			case "service", "services":
				tuple.Resource = "services"
			case "pod", "pods":
				tuple.Resource = "pods"
			default:
				tuple.Resource = "customresourcedefinitions"
			}
			cmd.Resource = tuple.String()
		}
	}
}
