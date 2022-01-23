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
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/k8sutils"
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
		Ctx:            logCtx,
		KubeconfigPath: cmd.KubeConfig,
		Namespace:      cmd.Namespace,
		Workloads:      []string{cmd.Resource},
	}
	if err = connect.InitClient(logCtx); err != nil {
		logger.Errorln("init client err, please make sure your kubeconfig is available !")
		return
	}
	if err = connect.Prepare(logCtx); err != nil {
		return
	}
	_ = remote.NewDHCPManager(connect.GetClientSet(), cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary(logCtx)
	GetOrGenerateConfigMapWatcher(connect.KubeconfigBytes, cmd.Namespace, connect.GetClientSet().CoreV1().RESTClient())
	switch cmd.Action {
	case command.Connect:
		var client *daemon_client.DaemonClient
		client, err = daemon_client.GetDaemonClient(true)
		if err != nil {
			return err
		}
		// pre-check if resource already in reversing mode
		if load, ok := GetReverseInfo().Load(util.GenerateKey(connect.KubeconfigBytes, connect.Namespace)); ok {
			if mac := load.(*status).getMacByResource(cmd.Resource); len(mac) != 0 {
				if mac == util.GetMacAddress().String() {
					err = fmt.Errorf("resource: %s is already in reversing mode by yourself", cmd.Resource)
				} else {
					err = fmt.Errorf("resource: %s is already in reversing mode by another one", cmd.Resource)
				}
				return
			}
		}

		// change to another cluster or namespace, clean all reverse
		if !connectInfo.IsEmpty() && !connectInfo.IsSame(connect.KubeconfigBytes, cmd.Namespace) {
			logger.Infof("you already connect to namespace: %s, switching to namespace: %s...\n",
				connectInfo.namespace, cmd.Namespace)
			clientset, err := util.GetClientSetByKubeconfigBytes(connectInfo.kubeconfigBytes)
			if err != nil {
				return err
			}
			path := k8sutils.GetOrGenKubeConfigPath(string(connectInfo.kubeconfigBytes))

			// cleanup all reverse
			logger.Infof("cleanup all old reverse resources...\n")
			if value, found := GetReverseInfo().Load(connectInfo.toKey()); found {
				connectOptions := &pkg.ConnectOptions{
					Ctx:            logCtx,
					KubeconfigPath: path,
					Namespace:      connectInfo.namespace,
				}
				if err = connectOptions.InitClient(context.TODO()); err != nil {
					break
				}
				for _, d := range value.(*status).reverse.LoadAndDeleteBelongToMeResources().KeySet() {
					_ = UpdateReverseConfigMap(clientset, connectInfo.namespace, d,
						func(r *ReverseTotal, record ReverseRecord) {
							r.RemoveRecord(record)
						})
					connectOptions.Workloads = []string{d}
					if err = connectOptions.RemoveInboundPod(); err != nil {
						logger.Errorf("delete reverse resource: %s in namespace: %s, error: %v\n", connectInfo.namespace, d, err)
					} else {
						logger.Infof("delete reverse resource: %s in namespace: %s successfully\n", connectInfo.namespace, d)
					}
				}
			}

			// disconnect from old cluster or namespace
			if err = UpdateConnect(clientset, connectInfo.namespace, func(list sets.String, item string) {
				list.Delete(item)
			}); err != nil {
				logger.Infof("error while remove connection info of namespace: %s\n", connectInfo.namespace)
			}

			logger.Infof("disconnecting from old namespace...\n")
			if r, err := client.SendSudoVPNOperateCommand(
				path, connectInfo.namespace, command.DisConnect, cmd.Resource); err == nil {
				if ok := transStreamToWriter(writer, r); !ok {
					return fmt.Errorf("failed to disconnect from old namespace: %s", connectInfo.namespace)
				}
			} else {
				return fmt.Errorf("failed to send disconnect request to sudo daemon")
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
		//}
		logger.Infof("connecting to new namespace...\n")
		if r, err := client.SendSudoVPNOperateCommand(
			cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource); err == nil {
			if ok := transStreamToWriter(writer, r); !ok {
				return fmt.Errorf("failed to connect to namespace: %s", cmd.Namespace)
			}
		} else {
			return err
		}
		logger.Infof("connected to new namespace\n")
		// reverse resource if needed
		if len(cmd.Resource) != 0 {
			logger.Infof("prepare to reverse resource: %s...\n", cmd.Resource)
			_ = UpdateReverseConfigMap(connect.GetClientSet(), cmd.Namespace, cmd.Resource, func(r *ReverseTotal, record ReverseRecord) {
				r.AddRecord(record)
			})
			if err = connect.DoReverse(logCtx); err != nil {
				logger.Infof("reverse resource: %s occours error, err: %v\n", cmd.Resource, err)
			} else {
				logger.Infof("reverse resource: %s successfully\n", cmd.Resource)
			}
		}
		return
	case command.Reconnect:
		var client *daemon_client.DaemonClient
		client, err = daemon_client.GetDaemonClient(true)
		if err != nil {
			return
		}
		if err = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
			list.Insert(address)
		}); err != nil {
			return
		}
		var r io.ReadCloser
		//err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource)
		r, err = client.SendSudoVPNOperateCommand(cmd.KubeConfig, cmd.Namespace, command.Connect, cmd.Resource)
		if err == nil {
			if ok := transStreamToWriter(writer, r); !ok {
				err = fmt.Errorf("failed to reconnect to namespace: %s", cmd.Namespace)
			}
		}
		if len(cmd.Resource) != 0 {
			return connect.DoReverse(context.TODO())
		}
		return
	case command.DisConnect:
		defer writer.Close()
		if len(cmd.Resource) != 0 {
			load, ok := GetReverseInfo().Load(util.GenerateKey(connect.KubeconfigBytes, connect.Namespace))
			if !ok {
				logger.Infof("can not found reverse info in namespace: %s, no need to cancel it\n", connect.Namespace)
				return nil
			}
			address := load.(*status).getMacByResource(cmd.Resource)
			// update reverse data immediately
			load.(*status).deleteByResource(cmd.Resource)
			_ = UpdateReverseConfigMap(connect.GetClientSet(), cmd.Namespace, cmd.Resource,
				func(r *ReverseTotal, record ReverseRecord) {
					record.MacAddress = address
					r.RemoveRecord(record)
				},
			)
			if err = connect.RemoveInboundPod(); err != nil {
				logger.Errorf("error while delete reverse pods, resource: %s in namespace: %s, error: %v",
					connect.Namespace, cmd.Resource, err)
			} else {
				logger.Infof("delete reverse pod, info: %s-%s secusefully",
					connect.Namespace, cmd.Resource)
			}

			// if cancel last reverse resources, needs to close connect
			if value, found := GetReverseInfo().Load(util.GenerateKey(connect.KubeconfigBytes, connect.Namespace)); found {
				if value.(*status).reverse.GetBelongToMeResources().Len() == 0 {
					_ = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
						list.Delete(address)
					})
					logger.Infof("have no reverse resource, disconnecting from namespace: %s...\n", cmd.Namespace)
					client, err := daemon_client.GetDaemonClient(true)
					if err != nil {
						return err
					}
					if r, err := client.SendSudoVPNOperateCommand(
						cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
						if ok = transStreamToWriter(writer, r); !ok {
							return fmt.Errorf("failed to disconnect from namespace: %s", cmd.Namespace)
						}
					}
				}
			}
			return
		} else {
			logger.Infof("disconnecting from namespace: %s\n", cmd.Namespace)
			_ = UpdateConnect(connect.GetClientSet(), cmd.Namespace, func(list sets.String, address string) {
				list.Delete(address)
			})
			value, loaded := GetReverseInfo().Load(util.GenerateKey(connect.KubeconfigBytes, connect.Namespace))
			if loaded {
				clientset, err := util.GetClientSetByKubeconfigBytes(value.(*status).kubeconfigBytes)
				if err != nil {
					break
				}
				path := k8sutils.GetOrGenKubeConfigPath(string(value.(*status).kubeconfigBytes))
				temp := &pkg.ConnectOptions{
					Ctx:            logCtx,
					KubeconfigPath: path,
					Namespace:      value.(*status).namespace,
					Workloads:      []string{},
				}
				if err = temp.InitClient(context.TODO()); err != nil {
					break
				}
				for _, resource := range value.(*status).reverse.LoadAndDeleteBelongToMeResources().KeySet() {
					_ = UpdateReverseConfigMap(clientset,
						value.(*status).namespace,
						resource,
						func(r *ReverseTotal, record ReverseRecord) {
							r.RemoveRecord(record)
						})
					temp.Workloads = []string{resource}
					_ = temp.RemoveInboundPod()
				}
			}
			var client *daemon_client.DaemonClient
			client, err = daemon_client.GetDaemonClient(true)
			if err != nil {
				return err
			}
			if r, err := client.SendSudoVPNOperateCommand(
				cmd.KubeConfig, cmd.Namespace, command.DisConnect, cmd.Resource); err == nil {
				if ok := transStreamToWriter(writer, r); !ok {
					return fmt.Errorf("failed to disconnect from namespace: %s", cmd.Namespace)
				}
			}
			return
		}
	default:
		return errors.New("Unsupported operation: %s" + string(cmd.Action))
	}
	// todo
	return
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

func UpdateConnect(clientSet *kubernetes.Clientset, namespace string, f func(connectedList sets.String, macAddress string)) error {
	//if !connectInfo.IsEmpty() {
	//	// todo
	//	return nil
	//}
	configMap, err := clientSet.CoreV1().ConfigMaps(namespace).Get(context.TODO(), util.TrafficManager, v1.GetOptions{})
	if err != nil {
		log.Warn(err)
		return err
	}
	t := FromStringToConnectInfo(configMap.Data[util.Connect])
	f(t.list, util.GetMacAddress().String())
	configMap.Data[util.Connect] = t.ToString()
	_, err = clientSet.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configMap, v1.UpdateOptions{})
	return err
}

// MacAddress1:deployment/test,service/test
// MacAddress2:deployment/hello,service/world

func init() {
	util.InitLogger(util.Debug)
}

// true: command execute no error, false: command occur error
func transStreamToWriter(writer io.Writer, r io.ReadCloser) bool {
	if r == nil {
		return false
	}
	defer func() { _ = r.Close() }()
	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if errors.Is(io.EOF, err) {
				return true
			}
			return false
		}
		if len(line) != 0 {
			if strings.Contains(string(line), util.EndSignOK) || strings.Contains(string(line), util.EndSignFailed) {
				return strings.Contains(string(line), util.EndSignOK)
			}
			_, _ = writer.Write(line)
			_, _ = writer.Write([]byte("\n"))
		}
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
