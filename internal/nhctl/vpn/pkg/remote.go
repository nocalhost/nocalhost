package pkg

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/util/podutils"
	"net"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"sort"
	"strings"
	"time"
)

func CreateOutboundRouterPodIfNecessary(clientset *kubernetes.Clientset, namespace string, serverIp *net.IPNet, nodeCIDR []*net.IPNet, logger *log.Logger) (string, error) {
	firstPod, i, err3 := polymorphichelpers.GetFirstPod(clientset.CoreV1(),
		namespace,
		fields.OneTermEqualSelector("app", util.TrafficManager).String(),
		time.Second*5,
		func(pods []*v1.Pod) sort.Interface {
			return sort.Reverse(podutils.ActivePods(pods))
		},
	)

	if err3 == nil && i != 0 && firstPod != nil {
		remote.UpdateRefCount(clientset, namespace, firstPod.Name, 1)
		return firstPod.Status.PodIP, nil
	}
	args := []string{
		"sysctl net.ipv4.ip_forward=1",
		"iptables -F",
		"iptables -P INPUT ACCEPT",
		"iptables -P FORWARD ACCEPT",
		fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o eth0 -j MASQUERADE", util.RouterIP.String()),
	}
	for _, ipNet := range nodeCIDR {
		args = append(args, fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o eth0 -j MASQUERADE", ipNet.String()))
	}
	args = append(args, fmt.Sprintf("nhctl vpn serve -L tcp://:10800 -L tun://:8421?net=%s --debug=true", serverIp.String()))

	t := true
	zero := int64(0)
	name := util.TrafficManager
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      map[string]string{"app": util.TrafficManager},
			Annotations: map[string]string{"ref-count": "1"},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyAlways,
			Containers: []v1.Container{
				{
					Name:    "vpn",
					Image:   _const.DefaultVPNImage,
					Command: []string{"/bin/sh", "-c"},
					Args:    []string{strings.Join(args, ";")},
					SecurityContext: &v1.SecurityContext{
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{
								"NET_ADMIN",
								//"SYS_MODULE",
							},
						},
						RunAsUser:  &zero,
						Privileged: &t,
					},
					Resources: v1.ResourceRequirements{
						Requests: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:    resource.MustParse("128m"),
							v1.ResourceMemory: resource.MustParse("256Mi"),
						},
						Limits: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:    resource.MustParse("256m"),
							v1.ResourceMemory: resource.MustParse("512Mi"),
						},
					},
					ImagePullPolicy: v1.PullAlways,
				},
			},
			PriorityClassName: "system-cluster-critical",
		},
	}
	_, err2 := clientset.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	if err2 != nil {
		logger.Errorln(err2)
		return "", err2
	}
	watch, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: name}))
	if err != nil {
		logger.Errorln(err)
		return "", err
	}
	defer watch.Stop()
	for {
		select {
		case e := <-watch.ResultChan():
			if podT, ok := e.Object.(*v1.Pod); ok && podT.Status.Phase == v1.PodRunning {
				return podT.Status.PodIP, nil
			}
		case <-time.Tick(time.Minute * 2):
			logger.Error("timeout")
			return "", errors.New("timeout")
		}
	}
}

func getController() Scalable {
	return nil
}

func CreateInboundPod(ctx context.Context, factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, workloads, virtualLocalIp, realRouterIP, virtualShadowIp, routes string) error {
	tuple, parsed, err2 := util.SplitResourceTypeName(workloads)
	if !parsed || err2 != nil {
		return errors.New("not need")
	}
	newName := ToInboundPodName(tuple.Resource, tuple.Name)
	util.DeletePod(clientset, namespace, newName)
	var sc Scalable
	switch strings.ToLower(tuple.Resource) {
	case "deployment", "deployments":
		sc = NewDeploymentController(factory, clientset, namespace, tuple.Name)
	case "statefulset", "statefulsets":
		sc = NewStatefulsetController(factory, clientset, namespace, tuple.Name)
	case "replicaset", "replicasets":
		sc = NewReplicasController(factory, clientset, namespace, tuple.Name)
	case "service", "services":
		sc = NewServiceController(factory, clientset, namespace, tuple.Name)
	case "pod", "pods":
		sc = NewPodController(factory, clientset, namespace, "pods", tuple.Name)
	default:
		sc = NewCustomResourceDefinitionController(factory, clientset, namespace, tuple.Resource, tuple.Name)
	}
	labels, ports, str, err2 := sc.ScaleToZero()
	util.GetLoggerFromContext(ctx).Infoln("scaling workloads to 0...")
	t := true
	zero := int64(0)
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newName,
			Namespace: namespace,
			Labels:    labels,
			// for restore
			Annotations: map[string]string{util.OriginData: str},
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyAlways,
			Containers: []v1.Container{
				{
					Name:    "vpn",
					Image:   _const.DefaultVPNImage,
					Command: []string{"/bin/sh", "-c"},
					Args: []string{
						"sysctl net.ipv4.ip_forward=1;" +
							"iptables -F;" +
							"iptables -P INPUT ACCEPT;" +
							"iptables -P FORWARD ACCEPT;" +
							"iptables -t nat -A PREROUTING -i eth0 -p tcp --dport 1:65535 -j DNAT --to " + virtualLocalIp + ":1-65535;" +
							"iptables -t nat -A POSTROUTING -p tcp -m tcp --dport 1:65535 -j MASQUERADE;" +
							"iptables -t nat -A PREROUTING -i eth0 -p udp --dport 1:65535 -j DNAT --to " + virtualLocalIp + ":1-65535;" +
							"iptables -t nat -A POSTROUTING -p udp -m udp --dport 1:65535 -j MASQUERADE;" +
							"nhctl vpn serve -L 'tun://0.0.0.0:8421/" + realRouterIP + ":8421?net=" + virtualShadowIp + "&route=" + routes + "' --debug=true",
					},
					SecurityContext: &v1.SecurityContext{
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{
								"NET_ADMIN",
								//"SYS_MODULE",
							},
						},
						RunAsUser:  &zero,
						Privileged: &t,
					},
					Resources: v1.ResourceRequirements{
						Requests: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:    resource.MustParse("128m"),
							v1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Limits: map[v1.ResourceName]resource.Quantity{
							v1.ResourceCPU:    resource.MustParse("256m"),
							v1.ResourceMemory: resource.MustParse("256Mi"),
						},
					},
					ImagePullPolicy: v1.PullAlways,
					// without helm, not set ports are works fine, but if using helm, must set this filed, otherwise
					// this pod will not add to service's endpoint
					Ports: ports,
				},
			},
			PriorityClassName: "system-cluster-critical",
		},
	}
	if _, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{}); err != nil {
		return err
	}
	watch, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: newName}))
	if err != nil {
		return err
	}
	defer watch.Stop()
	for {
		select {
		case e := <-watch.ResultChan():
			if p, ok := e.Object.(*v1.Pod); ok {
				util.GetLoggerFromContext(ctx).Infof("pods: %s is %s ...", p.Name, p.Status.Phase)
				if p.Status.Phase == v1.PodRunning {
					return nil
				}
			}
		case <-time.Tick(time.Minute * 5):
			util.GetLoggerFromContext(ctx).Infof("wait pods: %s to be ready timeout", newName)
			return errors.New(fmt.Sprintf("wait pods: %s to be ready timeout", newName))
		}
	}
}
