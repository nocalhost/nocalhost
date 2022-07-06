package daemon_handler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/k8sutils"
	"strings"
)

// HandleVPNOperate not sudo daemon, vpn controller
func HandleVPNOperate(cmd *command.VPNOperateCommand, writer io.WriteCloser) (err error) {
	logCtx := util.GetContextWithLogger(writer)
	logger := util.GetLoggerFromContext(logCtx)
	defer func() {
		if err != nil {
			logger.Errorln(err)
			logger.Errorln(util.EndSignFailed)
		} else {
			logger.Infoln(util.EndSignOK)
		}
		_ = writer.Close()
	}()
	connect := &pkg.ConnectOptions{
		Ctx:            logCtx,
		KubeconfigPath: cmd.KubeConfig,
		Namespace:      cmd.Namespace,
	}
	if err = connect.InitClient(logCtx); err != nil {
		logger.Errorln("init client err, please make sure your kubeconfig is available!")
		return
	}
	if err = connect.Prepare(logCtx); err != nil {
		return
	}
	_, err = remote.NewDHCPManager(connect.GetClientSet(), cmd.Namespace, &util.RouterIP).InitDHCPIfNecessary(logCtx)
	if err != nil {
		return err
	}
	GetOrGenerateConfigMapWatcher(connect.KubeconfigBytes, cmd.Namespace, connect.GetClientSet().CoreV1().RESTClient())

	if len(cmd.Resource) != 0 {
		// if controller is not nil
		object, errs := connect.GetUnstructuredObject(cmd.Resource)
		if errs != nil {
			return errs
		}
		workload := fmt.Sprintf("%s.%s.%s/%s",
			object.Mapping.Resource.Resource,
			object.Mapping.GroupVersionKind.Version,
			object.Mapping.GroupVersionKind.Group,
			object.Name)
		connect.Workloads = []string{workload}
		cmd.Resource = workload
		if own := object.Object.(*unstructured.Unstructured).GetOwnerReferences(); own != nil {
			return fmt.Errorf("controller is not nil, please connect to resource: %s/%s",
				own[0].Kind, own[0].Name)
		}
	}
	switch cmd.Action {
	case command.Connect:
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
		if !connectInfo.IsEmpty() && !connectInfo.IsSameUid(connect.Uid) {
			logger.Infof("switching from namespace: %s to namespace: %s...", connectInfo.namespace, cmd.Namespace)
			path := k8sutils.GetOrGenKubeConfigPath(string(connectInfo.kubeconfigBytes))
			if err = disconnectedFromNamespace(logCtx, writer, path, connectInfo.namespace); err != nil {
				return err
			}
		}

		// connect to new cluster or namespace
		if err = connectToNamespace(logCtx, writer, cmd.KubeConfig, cmd.Namespace); err != nil {
			return err
		}
		logger.Infof("connected to new namespace: %s", cmd.Namespace)
		// reverse resource if needed
		if len(cmd.Resource) != 0 {
			logger.Infof("prepare to reverse resource: %s...", cmd.Resource)
			_ = updateReverseConfigMap(cmd.KubeConfig, cmd.Namespace, []string{cmd.Resource}, add)
			if err = connect.DoReverse(logCtx); err != nil {
				logger.Infof("reverse resource: %s occours error, err: %v", cmd.Resource, err)
			} else {
				logger.Infof("reverse resource: %s successfully", cmd.Resource)
			}
		}
		return
	case command.Reconnect:
		if err = connectToNamespace(logCtx, writer, cmd.KubeConfig, cmd.Namespace); err != nil {
			return err
		}
		logger.Infof("connected to namespace: %s", cmd.Namespace)
		if len(cmd.Resource) != 0 {
			return connect.DoReverse(context.TODO())
		}
		return
	case command.DisConnect:
		defer func() { _ = writer.Close() }()
		if len(cmd.Resource) != 0 {
			logger.Infof("disconnecting to resource: %s", cmd.Resource)
			load, ok := GetReverseInfo().Load(util.GenerateKey(connect.KubeconfigBytes, connect.Namespace))
			if !ok {
				logger.Infof("can not found reverse info in namespace: %s, no need to cancel it", connect.Namespace)
				return nil
			}
			address := load.(*status).getMacByResource(cmd.Resource)
			// update reverse data immediately
			load.(*status).deleteByResource(cmd.Resource)
			_ = updateReverseConfigMap(cmd.KubeConfig, cmd.Namespace, []string{cmd.Resource},
				func(r *ReverseTotal, records ...*ReverseRecord) {
					for _, record := range records {
						record.MacAddress = address
						r.RemoveRecord(record)
					}
				},
			)
			if err = connect.RemoveInboundPod(); err != nil {
				logger.Error(err)
			}

			// if cancel last reverse resources, needs to close connect
			if value, found := GetReverseInfo().Load(util.GenerateKey(connect.KubeconfigBytes, connect.Namespace)); found {
				if value.(*status).reverse.GetBelongToMeResources().Len() != 0 {
					logger.Infof("disconnected to resource: %s", cmd.Resource)
					return
				}
			}
		}
		logger.Infof("disconnecting from namespace: %s", cmd.Namespace)
		return disconnectedFromNamespace(logCtx, writer, cmd.KubeConfig, cmd.Namespace)
	default:
		return fmt.Errorf("unsupported operation: %s", string(cmd.Action))
	}
}

var add = func(total *ReverseTotal, record ...*ReverseRecord) { total.AddRecord(record...) }

var remove = func(total *ReverseTotal, record ...*ReverseRecord) { total.RemoveRecord(record...) }

func updateReverseConfigMap(kubeconfigPath, namespace string, resource []string,
	f func(*ReverseTotal, ...*ReverseRecord)) error {
	options := pkg.ConnectOptions{KubeconfigPath: kubeconfigPath, Namespace: namespace}
	if err := options.InitClient(context.Background()); err != nil {
		return err
	}
	mapInterface := options.GetClientSet().CoreV1().ConfigMaps(namespace)
	cm, err := mapInterface.Get(context.Background(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	t := FromStringToReverseTotal(cm.Data[util.REVERSE])
	for _, s := range resource {
		f(t, NewReverseRecordWithWorkloads(s))
	}
	_, err = mapInterface.Patch(
		context.Background(),
		util.TrafficManager,
		types.MergePatchType,
		[]byte(fmt.Sprintf("{\"data\":{\"%s\":\"%s\"}}", util.REVERSE, t.ToString())),
		metav1.PatchOptions{},
	)
	return err
}

var deleteFunc = func(connectedList *ConnectTotal, macAddress string) { connectedList.list.Delete(macAddress) }

var insertFunc = func(connectedList *ConnectTotal, macAddress string) { connectedList.list.Insert(macAddress) }

func updateConnectConfigMap(mapInterface coreV1.ConfigMapInterface, f func(*ConnectTotal, string)) error {
	cm, err := mapInterface.Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		return err
	}
	t := FromStringToConnectInfo(cm.Data[util.Connect])
	f(t, util.GetMacAddress().String())
	_, err = mapInterface.Patch(
		context.Background(),
		util.TrafficManager,
		types.MergePatchType,
		[]byte(fmt.Sprintf("{\"data\":{\"%s\":\"%s\"}}", util.Connect, t.ToString())),
		metav1.PatchOptions{},
	)
	return err
}

// MacAddress1:deployment/test,service/test
// MacAddress2:deployment/hello,service/world

func init() {
	util.InitLogger(util.Debug)
}

// true: command execute no error, false: command occur error
func transStreamToWriter(r io.Reader, writer ...io.Writer) bool {
	if r == nil {
		return false
	}
	w := io.MultiWriter(writer...)
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
			_, _ = w.Write(line)
			_, _ = w.Write([]byte("\n"))
		}
	}
}

func connectToNamespace(ctx context.Context, writer io.WriteCloser, kubeconfigPath, namespace string) error {
	if !util.IsPortListening(daemon_common.SudoDaemonPort) {
		return errors.New("sudo daemon is not running")
	}
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return err
	}
	logger := util.GetLoggerFromContext(ctx)
	options := pkg.ConnectOptions{KubeconfigPath: kubeconfigPath, Namespace: namespace}
	if err = options.InitClient(ctx); err != nil {
		return err
	}
	if err = updateConnectConfigMap(options.GetClientSet().CoreV1().ConfigMaps(namespace), insertFunc); err != nil {
		return err
	}
	logger.Infof("connecting to new namespace: %s...", namespace)
	return client.SendSudoVPNOperateCommand(kubeconfigPath, namespace, command.Connect, func(r io.Reader) error {
		if ok := transStreamToWriter(r, writer); !ok {
			return fmt.Errorf("failed to connect to namespace: %s", namespace)
		}
		return nil
	})
}

func disconnectedFromNamespace(ctx context.Context, writer io.WriteCloser, kubeconfigPath, namespace string) error {
	logger := util.GetLoggerFromContext(ctx)
	kubeconfigBytes, _ := ioutil.ReadFile(kubeconfigPath)
	var err error
	// cleanup all reverse
	logger.Infof("cleanup all old reverse resources...")
	options := &pkg.ConnectOptions{
		Ctx:            ctx,
		KubeconfigPath: kubeconfigPath,
		Namespace:      namespace,
	}
	if err = options.InitClient(ctx); err != nil {
		return err
	}
	if value, found := GetReverseInfo().Load(util.GenerateKey(kubeconfigBytes, namespace)); found {
		set := value.(*status).reverse.LoadAndDeleteBelongToMeResources().KeySet()
		_ = updateReverseConfigMap(kubeconfigPath, namespace, set, remove)
		// remove inbound pod
		options.Workloads = set
		if err := options.RemoveInboundPod(); err != nil {
			logger.Error(err)
		}
	}

	// disconnect from old cluster or namespace
	if err = updateConnectConfigMap(options.GetClientSet().CoreV1().ConfigMaps(namespace), deleteFunc); err != nil {
		logger.Infof("error while remove connection info of namespace: %s", namespace)
	}
	if !util.IsPortListening(daemon_common.SudoDaemonPort) {
		return errors.New("sudo daemon is not running")
	}
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		return err
	}
	logger.Infof("disconnecting from namespace: %s...", namespace)
	if err = client.SendSudoVPNOperateCommand(kubeconfigPath, namespace, command.DisConnect, func(r io.Reader) error {
		if ok := transStreamToWriter(r, writer); !ok {
			return fmt.Errorf("failed to disconnect from namespace: %s", namespace)
		}
		return nil
	}); err == nil {
		logger.Infof("disconnected from namespace: %s", namespace)
		return nil
	}
	return err
}
