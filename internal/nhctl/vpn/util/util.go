/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	dockerterm "github.com/moby/term"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"io"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	json2 "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	runtimeresource "k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_common"
	"regexp"
	"strconv"
	"time"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/cmd/exec"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func GetAvailableUDPPortOrDie() int {
	address, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:0", "0.0.0.0"))
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.ListenUDP("udp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	return listener.LocalAddr().(*net.UDPAddr).Port
}

func PortForwardPod(
	config *rest.Config,
	clientset *rest.RESTClient,
	podName,
	namespace,
	portPair string,
	readyChan chan struct{},
	stopChan <-chan struct{},
) error {
	url := clientset.
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").
		//Timeout(time.Second * 30).
		URL()
	transport, upgrader, err := spdy.RoundTripperFor(config)
	if err != nil {
		log.Error(err)
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	p := []string{portPair}
	forwarder, err := NewOnAddresses(dialer, []string{"0.0.0.0"}, p, stopChan, readyChan, os.Stdout, os.Stderr)
	if err != nil {
		log.Error(err)
		return err
	}

	if err = forwarder.ForwardPorts(); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func GetTopController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, workload string) (controller ResourceTupleWithScale) {
	object, err := GetUnstructuredObject(factory, namespace, workload)
	if err != nil {
		return
	}
	asSelector, _ := metav1.LabelSelectorAsSelector(GetLabelSelector(object.Object))
	podList, _ := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: asSelector.String(),
	})
	if len(podList.Items) == 0 {
		return
	}
	of := metav1.GetControllerOf(&podList.Items[0])
	for of != nil {
		object, err = GetUnstructuredObject(factory, namespace, fmt.Sprintf("%s/%s", of.Kind, of.Name))
		if err != nil {
			return
		}
		controller.Resource = strings.ToLower(of.Kind) + "s"
		controller.Name = of.Name
		of = GetOwnerReferences(object.Object)
	}
	return
}

func UpdateReplicasScale(clientset *kubernetes.Clientset, namespace string, controller ResourceTupleWithScale) error {
	err := retry.OnError(
		retry.DefaultRetry,
		func(err error) bool { return err != nil },
		func() error {
			result := &autoscalingv1.Scale{}
			err := clientset.AppsV1().RESTClient().Put().
				Namespace(namespace).
				Resource(controller.Resource).
				Name(controller.Name).
				SubResource("scale").
				VersionedParams(&metav1.UpdateOptions{}, scheme.ParameterCodec).
				Body(&autoscalingv1.Scale{
					ObjectMeta: metav1.ObjectMeta{
						Name:      controller.Name,
						Namespace: namespace,
					},
					Spec: autoscalingv1.ScaleSpec{
						Replicas: int32(controller.Scale),
					},
				}).
				Do(context.Background()).
				Into(result)
			return err
		})
	if err != nil {
		log.Errorf("update scale: %s-%s's replicas to %d failed, error: %v", controller.Resource, controller.Name, controller.Scale, err)
	}
	return err
}

func Shell(clientset *kubernetes.Clientset, restclient *rest.RESTClient, config *rest.Config, podName, namespace, cmd string) (string, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})

	if err != nil {
		return "", err
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		err = fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
		return "", err
	}
	containerName := pod.Spec.Containers[0].Name
	stdin, _, stderr := dockerterm.StdStreams()

	stdoutBuf := bytes.NewBuffer(nil)
	stdout := io.MultiWriter(stdoutBuf)
	option := exec.StreamOptions{
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		IOStreams:     genericclioptions.IOStreams{In: stdin, Out: stdout, ErrOut: stderr},
	}
	executor := &exec.DefaultRemoteExecutor{}
	// ensure we can recover the terminal while attached
	tt := option.SetupTTY()

	var sizeQueue remotecommand.TerminalSizeQueue
	if tt.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = tt.MonitorSize(tt.GetSize())

		// unset p.Err if it was previously set because both stdout and stderr go over p.Out when tty is
		// true
		option.ErrOut = nil
	}

	fn := func() error {
		req := restclient.Post().
			Resource("pods").
			Name(pod.Name).
			Namespace(pod.Namespace).
			SubResource("exec").
			VersionedParams(&v1.PodExecOptions{
				Container: containerName,
				Command:   []string{"sh", "-c", cmd},
				Stdin:     option.Stdin,
				Stdout:    option.Out != nil,
				Stderr:    option.ErrOut != nil,
				TTY:       tt.Raw,
			}, scheme.ParameterCodec)
		return executor.Execute("POST", req.URL(), config, option.In, option.Out, option.ErrOut, tt.Raw, sizeQueue)
	}

	err = tt.Safe(fn)
	return strings.TrimRight(stdoutBuf.String(), "\n"), err
}

func ShellWithStream(clientset *kubernetes.Clientset, restclient *rest.RESTClient, config *rest.Config, podName, namespace, cmd string, writer io.Writer) error {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})

	if err != nil {
		return err
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		err = fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
		return err
	}
	containerName := pod.Spec.Containers[0].Name
	stdin, _, stderr := dockerterm.StdStreams()

	StreamOptions := exec.StreamOptions{
		Namespace:     namespace,
		PodName:       podName,
		ContainerName: containerName,
		IOStreams:     genericclioptions.IOStreams{In: stdin, Out: writer, ErrOut: stderr},
	}
	Executor := &exec.DefaultRemoteExecutor{}
	// ensure we can recover the terminal while attached
	tt := StreamOptions.SetupTTY()

	var sizeQueue remotecommand.TerminalSizeQueue
	if tt.Raw {
		// this call spawns a goroutine to monitor/update the terminal size
		sizeQueue = tt.MonitorSize(tt.GetSize())

		// unset p.Err if it was previously set because both stdout and stderr go over p.Out when tty is
		// true
		StreamOptions.ErrOut = nil
	}
	return tt.Safe(func() error {
		req := restclient.Post().
			Resource("pods").
			Name(pod.Name).
			Namespace(pod.Namespace).
			SubResource("exec")
		req.VersionedParams(&v1.PodExecOptions{
			Container: containerName,
			Command:   []string{"sh", "-c", cmd},
			Stdin:     StreamOptions.Stdin,
			Stdout:    StreamOptions.Out != nil,
			Stderr:    StreamOptions.ErrOut != nil,
			TTY:       tt.Raw,
		}, scheme.ParameterCodec)
		return Executor.Execute("POST", req.URL(), config, StreamOptions.In, StreamOptions.Out, StreamOptions.ErrOut, tt.Raw, sizeQueue)
	})
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func GetUnstructuredObject(f cmdutil.Factory, namespace string, workloads string) (*runtimeresource.Info, error) {
	do := f.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().AllNamespaces(false).
		ResourceTypeOrNameArgs(true, workloads).
		ContinueOnError().
		Latest().
		Flatten().
		TransformRequests(func(req *rest.Request) { req.Param("includeObject", "Object") }).
		Do()
	if err := do.Err(); err != nil {
		log.Warn(err)
		return nil, err
	}
	infos, err := do.Infos()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if len(infos) == 0 {
		return nil, errors.New("Not found")
	}
	return infos[0], err
}

func GetAndConsumeControllerObject(f cmdutil.Factory, namespace string, label labels.Selector, ff func(*unstructured.Unstructured)) ([]*resource.Info, error) {
	do := f.NewBuilder().
		Unstructured().
		NamespaceParam(namespace).DefaultNamespace().AllNamespaces(false).
		ResourceTypeOrNameArgs(true, strings.Join([]string{"deployments", "statefulset", "replicaset"}, ",")).
		ContinueOnError().
		LabelSelector(label.String()).
		Latest().
		Flatten().
		TransformRequests(func(req *rest.Request) { req.Param("includeObject", "Object") }).
		Do()
	if err := do.Err(); err != nil {
		return nil, err
	}
	infos, err := do.Infos()
	if err != nil {
		return nil, err
	}
	if ff != nil {
		for _, info := range infos {
			if o, ok := info.Object.(*unstructured.Unstructured); ok {
				ff(o)
			}
		}
	}
	return infos, nil
}

func GetLabelSelector(object k8sruntime.Object) *metav1.LabelSelector {
	u := object.(*unstructured.Unstructured)
	stringMap, _, err := unstructured.NestedStringMap(u.Object, "spec", "selector")
	if err == nil && len(stringMap) != 0 {
		return &metav1.LabelSelector{MatchLabels: stringMap}
	}
	return &metav1.LabelSelector{MatchLabels: u.GetLabels()}
}

func GetPorts(object k8sruntime.Object) []v1.ContainerPort {
	defer func() {
		if err := recover(); err != nil {
			log.Errorln(err)
		}
	}()
	var result []v1.ContainerPort
	replicasetPortPrinter, _ := printers.NewJSONPathPrinter("{.spec.template.spec.containers[0].ports}")
	servicePortPrinter, _ := printers.NewJSONPathPrinter("{.spec.ports}")
	buf := bytes.NewBuffer([]byte{})
	err := replicasetPortPrinter.PrintObj(object, buf)
	if err != nil {
		_ = servicePortPrinter.PrintObj(object, buf)
		var ports []v1.ServicePort
		_ = json2.Unmarshal([]byte(buf.String()), &ports)
		for _, port := range ports {
			val := port.TargetPort.IntVal
			if val == 0 {
				val = port.Port
			}
			result = append(result, v1.ContainerPort{
				Name:          port.Name,
				ContainerPort: val,
				Protocol:      port.Protocol,
			})
		}
	} else {
		_ = json2.Unmarshal([]byte(buf.String()), &result)
	}
	return result
}

func GetOwnerReferences(object k8sruntime.Object) *metav1.OwnerReference {
	refs := object.(metav1.Object).GetOwnerReferences()
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	return nil
}

func GetScale(object k8sruntime.Object) int {
	defer func() {
		if err := recover(); err != nil {
			log.Errorln(err)
		}
	}()
	printer, _ := printers.NewJSONPathPrinter("{.spec.replicas}")
	buf := bytes.NewBuffer([]byte{})
	if err := printer.PrintObj(object, buf); err != nil {
		return 0
	}
	if atoi, err := strconv.Atoi(buf.String()); err == nil {
		return atoi
	}
	return 0
}

func DeletePod(clientset *kubernetes.Clientset, namespace, podName string) error {
	zero := int64(0)
	err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), podName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
	})
	if k8serrors.IsNotFound(err) {
		log.Infof("not found shadow pod: %s, no need to delete it", podName)
		return nil
	}
	return err
}

type ResourceTuple struct {
	Resource string
	Name     string
}

func (r ResourceTuple) String() string {
	return fmt.Sprintf("%s/%s", r.Resource, r.Name)
}

type ResourceTupleWithScale struct {
	Resource string
	Name     string
	Scale    int
}

// splitResourceTypeName handles type/name resource formats and returns a resource tuple
// (empty or not), whether it successfully found one, and an error
func SplitResourceTypeName(s string) (ResourceTuple, bool, error) {
	if !strings.Contains(s, "/") {
		return ResourceTuple{}, false, nil
	}
	seg := strings.Split(s, "/")
	if len(seg) != 2 {
		return ResourceTuple{}, false, fmt.Errorf("arguments in resource/name form may not have more than one slash")
	}
	resource, name := seg[0], seg[1]
	if len(resource) == 0 || len(name) == 0 || len(runtimeresource.SplitResourceArgument(resource)) != 1 {
		return ResourceTuple{}, false, fmt.Errorf("arguments in resource/name form must have a single resource and name")
	}
	return ResourceTuple{Resource: resource, Name: name}, true, nil
}

func DeleteConfigMap(clientset *kubernetes.Clientset, namespace, configMapName string) {
	_ = clientset.CoreV1().ConfigMaps(namespace).Delete(context.Background(), configMapName, metav1.DeleteOptions{})
}

func BytesToInt(b []byte) uint32 {
	buffer := bytes.NewBuffer(b)
	var u uint32
	if err := binary.Read(buffer, binary.BigEndian, &u); err != nil {
		log.Warn(err)
	}
	return u
}

func IsPortListening(port int) bool {
	listener, err := net.Listen("tcp4", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
	if err != nil {
		return true
	} else {
		listener.Close()
		return false
	}
}

func IsSudoDaemonServing() bool {
	if !IsPortListening(daemon_common.SudoDaemonPort) {
		return false
	}
	if _, err := daemon_client.GetDaemonClient(true); err != nil {
		return false
	}
	return true
}

var mac net.HardwareAddr

func GetMacAddress() net.HardwareAddr {
	if mac != nil {
		return mac
	}
	if interfaces, err := net.Interfaces(); err == nil {
		for _, ifc := range interfaces {
			if ifc.HardwareAddr != nil {
				if ifc.Flags&net.FlagUp|net.FlagMulticast|net.FlagBroadcast ==
					net.FlagUp|net.FlagMulticast|net.FlagBroadcast {
					return ifc.HardwareAddr
				}
			}
		}
		for _, ifc := range interfaces {
			if ifc.HardwareAddr != nil {
				if ifc.Flags&net.FlagUp == net.FlagUp {
					return ifc.HardwareAddr
				}
			}
		}
	}
	return net.HardwareAddr{0x00, 0x00, 0x5e, 0x00, 0x53, 0x01}
}

func MergeOrReplaceAnnotation(c rest.Interface, namespace, resource, name, k, v string) error {
	marshal, _ := json.Marshal([]PatchOperation{{
		Op:    "replace",
		Path:  "/metadata/annotations/" + k,
		Value: v,
	}})
	return c.
		Patch(types.JSONPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Body(marshal).
		Do(context.TODO()).
		Error()
}

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func GetContextWithLogger(writer io.WriteCloser) context.Context {
	ctx, _ := context.WithCancel(context.WithValue(context.TODO(), "logger", NewLogger(writer)))
	return ctx
}

func NewLogger(writer io.WriteCloser) *log.Logger {
	return &log.Logger{
		Out:          writer,
		Formatter:    &Format{},
		Hooks:        make(log.LevelHooks),
		Level:        log.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: true,
	}
}

func GetLoggerFromContext(ctx context.Context) *log.Logger {
	if l := ctx.Value("logger"); l != nil {
		return l.(*log.Logger)
	} else {
		return NewLogger(os.Stdout)
	}
}

func GetClientSetByKubeconfigBytes(kubeconfigBytes []byte) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

var reg, _ = regexp.Compile("[\\s\\t\\n\\r]")

// GenerateKey todo jetbrains rename will modify cluster name, it will make this function invalid
func GenerateKey(kubeconfigBytes []byte, namespace string) string {
	h := sha1.New()
	h.Write(reg.ReplaceAll(kubeconfigBytes, []byte("")))
	return string(h.Sum([]byte(namespace)))
}

func Ping(targetIP string) (bool, error) {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false, err
	}
	defer conn.Close()

	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("HELLO-R-U-THERE"),
		},
	}
	data, err := message.Marshal(nil)
	if err != nil {
		return false, nil
	}
	if _, err = conn.WriteTo(data, &net.IPAddr{IP: net.ParseIP(targetIP)}); err != nil {
		return false, err
	}

	rb := make([]byte, 1500)
	n, _, err := conn.ReadFrom(rb)
	if err != nil {
		return false, err
	}
	rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), rb[:n])
	if err != nil {
		return false, err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return true, nil
	default:
		return false, nil
	}
}

func WaitPortToBeFree(port int, timeout time.Duration) error {
	for {
		select {
		case <-time.Tick(timeout):
			return fmt.Errorf("wait port %v to be free timeout", port)
		case <-time.Tick(time.Second * 1):
			if !IsPortListening(port) {
				return nil
			}
		}
	}
}
