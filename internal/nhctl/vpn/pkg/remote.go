/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
	"time"
)

func createOutboundRouterPodIfNecessary(
	clientset *kubernetes.Clientset,
	namespace string,
	serverIP *net.IPNet,
	podCIDR []*net.IPNet,
	logger *log.Logger,
) (string, error) {
	routerPod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err == nil && routerPod.DeletionTimestamp == nil {
		remote.UpdateRefCount(clientset, namespace, routerPod.Name, 1)
		logger.Infoln("traffic manager already exist, not need to create it")
		return routerPod.Status.PodIP, nil
	}
	logger.Infoln("try to create traffic manager...")
	args := []string{
		"sysctl net.ipv4.ip_forward=1",
		"iptables -F",
		"iptables -P INPUT ACCEPT",
		"iptables -P FORWARD ACCEPT",
		fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o eth0 -j MASQUERADE", util.RouterIP.String()),
	}
	for _, ipNet := range podCIDR {
		args = append(args, fmt.Sprintf("iptables -t nat -A POSTROUTING -s %s -o eth0 -j MASQUERADE", ipNet.String()))
	}
	args = append(args, fmt.Sprintf("nhctl vpn serve -L tcp://:10800 -L tun://:8421?net=%s --debug=true", serverIP.String()))

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
	pods, err := clientset.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	if pods.Status.Phase == v1.PodRunning {
		return pods.Status.PodIP, nil
	}
	watch, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: name}))
	if err != nil {
		return "", err
	}
	defer watch.Stop()
	var phase v1.PodPhase
	for {
		select {
		case e := <-watch.ResultChan():
			if podT, ok := e.Object.(*v1.Pod); ok {
				if phase != podT.Status.Phase {
					logger.Infof("traffic manager is %s...", podT.Status.Phase)
				}
				if podT.Status.Phase == v1.PodRunning {
					return podT.Status.PodIP, nil
				}
				phase = podT.Status.Phase
			}
		case <-time.Tick(time.Minute * 5):
			return "", errors.New("wait for pod traffic manager to be ready timeout")
		}
	}
}

func getController() Scalable {
	return nil
}

func CreateInboundPod(
	ctx context.Context,
	factory cmdutil.Factory,
	clientset *kubernetes.Clientset,
	namespace,
	workloads,
	localTunIP,
	trafficManagerIP,
	shadowTunIP,
	routes string,
) error {
	var sc Scalable
	sc, err := getHandler(factory, clientset, namespace, workloads)
	if err != nil {
		return err
	}
	util.GetLoggerFromContext(ctx).Infoln("scaling workloads to 0...")
	labels, annotations, ports, str, err := sc.ScaleToZero()
	if err != nil {
		util.GetLoggerFromContext(ctx).Errorf("scale workloads to 0 failed, error: %v\n", err)
		return err
	}
	get, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), sc.ToInboundPodName(), metav1.GetOptions{})
	if get != nil {
		if o := get.Annotations[util.OriginData]; len(o) != 0 {
			str = o
		}
		zero := int64(0)
		_ = clientset.CoreV1().Pods(namespace).Delete(
			context.TODO(), sc.ToInboundPodName(), metav1.DeleteOptions{GracePeriodSeconds: &zero},
		)
	}

	annotations[util.OriginData] = str
	delete(annotations, "deployment.kubernetes.io/revision")
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")

	t := true
	zero := int64(0)
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sc.ToInboundPodName(),
			Namespace: namespace,
			Labels:    labels,
			// for restore
			Annotations: annotations,
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
							"iptables -t nat -A PREROUTING ! -p icmp -j DNAT --to " + localTunIP + ";" +
							"iptables -t nat -A POSTROUTING ! -p icmp -j MASQUERADE;" +
							"sysctl -w net.ipv4.conf.all.route_localnet=1;" +
							"iptables -t nat -A OUTPUT -o lo ! -p icmp -j DNAT --to-destination " + localTunIP + ";" +
							"nhctl vpn serve -L 'tun://0.0.0.0:8421/" + trafficManagerIP + ":8421?net=" + shadowTunIP + "&route=" + routes + "' --debug=true",
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
	newName := sc.ToInboundPodName()
	if _, err = clientset.CoreV1().Pods(namespace).Create(context.TODO(), &pod, metav1.CreateOptions{}); err != nil {
		return err
	}
	watch, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: newName}))
	if err != nil {
		return err
	}
	defer watch.Stop()
	var s v1.PodPhase
	for {
		select {
		case e := <-watch.ResultChan():
			if p, ok := e.Object.(*v1.Pod); ok {
				if p.Status.Phase != s {
					s = p.Status.Phase
					util.GetLoggerFromContext(ctx).Infof("pods: %s is %s...", p.Name, p.Status.Phase)
				}
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
