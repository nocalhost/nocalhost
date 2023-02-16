/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/vpn/pkg/handler"
	"nocalhost/internal/nhctl/vpn/remote"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
)

func createOutboundRouterPodIfNecessary(
	clientset *kubernetes.Clientset,
	ns string,
	serverIP *net.IPNet,
	podCIDR []*net.IPNet,
	logger *log.Logger,
) (net.IP, error) {
	routerPod, err := clientset.CoreV1().Pods(ns).Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err == nil && routerPod.DeletionTimestamp == nil {
		remote.UpdateRefCount(clientset, ns, routerPod.Name, 1)
		logger.Infoln("traffic manager already exist, not need to create it")
		return net.ParseIP(routerPod.Status.PodIP), nil
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
	args = append(args,
		fmt.Sprintf("nhctl vpn serve -L tcp://:10800 -L tun://:8421?net=%s --debug=true", serverIP.String()))

	t := true
	zero := int64(0)
	name := util.TrafficManager
	pod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   ns,
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
					// TODO: get image pull policy from config
					ImagePullPolicy: v1.PullIfNotPresent,
				},
			},
			PriorityClassName: "system-cluster-critical",
		},
	}
	pods, err := clientset.CoreV1().Pods(ns).Create(context.TODO(), &pod, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	if pods.Status.Phase == v1.PodRunning {
		return net.ParseIP(pods.Status.PodIP), nil
	}
	w, err := clientset.CoreV1().Pods(ns).Watch(context.TODO(), metav1.SingleObject(metav1.ObjectMeta{Name: name}))
	if err != nil {
		return nil, err
	}
	defer w.Stop()
	var phase v1.PodPhase
	for {
		select {
		case e := <-w.ResultChan():
			if e.Type == watch.Deleted {
				return nil, errors.New("traffic manager is deleted")
			}
			if podT, ok := e.Object.(*v1.Pod); ok {
				if phase != podT.Status.Phase {
					logger.Infof("traffic manager is %s...", podT.Status.Phase)
				}
				if podT.Status.Phase == v1.PodRunning {
					return net.ParseIP(podT.Status.PodIP), nil
				}
				phase = podT.Status.Phase
			}
		case <-time.Tick(time.Minute * 5):
			return nil, errors.New("wait for pod traffic manager to be ready timeout")
		}
	}
}

// CreateInboundPod
// 1, set replicset to 1
// 2, backup origin manifest to workloads annotation
// 3, patch a new sidecar
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
	var sc handler.Handler
	sc, err := getHandler(factory, clientset, namespace, workloads, &handler.PodRouteConfig{
		LocalTunIP:           localTunIP,
		InboundPodTunIP:      shadowTunIP,
		TrafficManagerRealIP: trafficManagerIP,
		Route:                routes,
	})
	if err != nil {
		return err
	}
	util.GetLoggerFromContext(ctx).Infoln("inject vpn sidecar ...")
	err = sc.InjectVPNContainer()
	if err != nil {
		util.GetLoggerFromContext(ctx).Errorf("inject vpn sidecar failed, error: %v\n", err)
		return err
	}
	util.GetLoggerFromContext(ctx).Infoln("inject vpn sidecar ok")
	return nil
}
