/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"

	"nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
)

type authConfig struct {
	hostConfig *rest.Config
}

var _ AuthConfig = &authConfig{}

func (a *authConfig) Get(vc *v1alpha1.VirtualCluster) (string, error) {
	name := vc.GetReleaseName()
	namespace := vc.GetNamespace()
	if a.hostConfig == nil {
		return "", errors.New("hostConfig is nil")
	}
	clientSet, err := kubernetes.NewForConfig(a.hostConfig)
	if err != nil {
		return "", err
	}

	// 1. get pod
	pod, err := GetVClusterPod(name, namespace, 5*time.Second, 30*time.Second, clientSet)
	if err != nil {
		return "", err
	}

	// 2. get kubeconfig
	OriginalKubeConfig, err := GetVClusterKubeConfigFromPod(pod, a.hostConfig, clientSet)
	if err != nil {
		return "", err
	}

	// 3. get address and port for kube-apiserver
	kubeConfig, err := clientcmd.Load(OriginalKubeConfig)
	if err != nil {
		return "", errors.WithStack(err)
	}
	addr, port, err := GetAddrAndPort(name, namespace, 2*time.Second, 10*time.Second, clientSet)

	// 4. set address into kubeconfig
	newClusterName := vc.GetSpaceName()
	hostClusterName := vc.GetHostClusterName()
	if hostClusterName == "" {
		hostClusterName = "vcluster"
	}
	newCtxName := hostClusterName + "/" + newClusterName
	newClusterName = newCtxName

	newCtx := kubeConfig.Contexts[kubeConfig.CurrentContext]
	if newCtx == nil {
		return "", errors.New("vcluster kubeconfig context not found")
	}
	newCluster := kubeConfig.Clusters[newCtx.Cluster]

	if addr != "" && port != "" {
		newCluster.Server = fmt.Sprintf("https://%s:%s", addr, port)
	} else {
		newCluster.Server = fmt.Sprintf("https://%s:%s", "127.0.0.1", "8443")
	}
	newCluster.InsecureSkipTLSVerify = true
	newCluster.CertificateAuthorityData = nil

	newCtx.Cluster = newClusterName

	kubeConfig.CurrentContext = newCtxName
	kubeConfig.Contexts = map[string]*api.Context{
		newCtxName: newCtx,
	}
	kubeConfig.Clusters = map[string]*api.Cluster{
		newClusterName: newCluster,
	}

	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(out), nil
}

func GetVClusterPod(name, namespace string, interval, timeout time.Duration, c kubernetes.Interface) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	options := metav1.ListOptions{LabelSelector: fmt.Sprintf("app=vcluster,release=%s", name)}
	if err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pods, err := c.CoreV1().Pods(namespace).List(context.TODO(), options)
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		pod = pods.Items[0]
		for _, p := range pods.Items {
			if p.CreationTimestamp.Unix() > pod.CreationTimestamp.Unix() {
				pod = p
			}
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return false, errors.Errorf(
				"the pod current phase is %s", pod.Status.Phase)
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, errors.Errorf("timeout to get pod %s/%s", namespace, name)
	}
	return &pod, nil
}

func GetVClusterKubeConfigFromPod(pod *corev1.Pod, config *rest.Config, c kubernetes.Interface) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	req := c.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: "syncer",
		Command:   []string{"cat", "/root/.kube/config"},
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if err := exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	}); err != nil {
		return nil, errors.WithStack(err)
	}
	return stdout.Bytes(), nil
}

func GetAddrAndPort(name, namespace string, interval, timeout time.Duration, c kubernetes.Interface) (string, string, error) {
	var addr, port string
	if err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		svc, err := c.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if svc.Spec.Type == corev1.ServiceTypeNodePort {
			nodes, err := c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return false, nil
			}
			if len(nodes.Items) == 0 {
				return false, nil
			}
			address := make(map[corev1.NodeAddressType]string)
			for _, node := range nodes.Items {
				for _, addressType := range node.Status.Addresses {
					if _, ok := address[addressType.Type]; ok {
						continue
					} else if addressType.Address != "" {
						address[addressType.Type] = addressType.Address
					}
				}
			}
			if a, ok := address[corev1.NodeExternalIP]; ok {
				addr = a
			} else {
				addr = address[corev1.NodeInternalIP]
			}

			ports := svc.Spec.Ports
			for _, p := range ports {
				if p.TargetPort.String() == "8443" {
					port = strconv.Itoa(int(p.NodePort))
				}
			}
		}

		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			if svc.Status.LoadBalancer.Ingress[0].Hostname != "" {
				addr = svc.Status.LoadBalancer.Ingress[0].Hostname
			} else if svc.Status.LoadBalancer.Ingress[0].IP != "" {
				addr = svc.Status.LoadBalancer.Ingress[0].IP
			}
			ports := svc.Spec.Ports
			for _, p := range ports {
				if p.TargetPort.String() == "8443" {
					port = strconv.Itoa(int(p.Port))
				}
			}
			if addr == "" || port == "" {
				return false, nil
			}
		}
		return true, nil
	}); err != nil {
		return "", "", errors.Errorf("cannot get service %s/%s", namespace, name)
	}
	return addr, port, nil
}

func NewAuthConfig(hostConfig *rest.Config) AuthConfig {
	return &authConfig{
		hostConfig: hostConfig,
	}
}
