/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"bytes"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"sort"
	"strconv"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

type authConfig struct {
	hostConfig *rest.Config
}

var _ AuthConfig = &authConfig{}

func (a *authConfig) Get(name, namespace string) (string, error) {
	if a.hostConfig == nil {
		return "", errors.New("hostConfig is nil")
	}
	clientSet, err := kubernetes.NewForConfig(a.hostConfig)
	if err != nil {
		return "", err
	}

	options := metav1.ListOptions{LabelSelector: fmt.Sprintf("app=vcluster,release=%s", name)}
	var pod corev1.Pod
	if err = wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		pods, err := clientSet.CoreV1().Pods(namespace).List(context.TODO(), options)
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			return false, nil
		}
		sort.Slice(pods.Items, func(i, j int) bool {
			return pods.Items[i].CreationTimestamp.Unix() > pods.Items[j].CreationTimestamp.Unix()
		})
		pod = pods.Items[0]
		for _, p := range pods.Items {
			if p.CreationTimestamp.Unix() > pod.CreationTimestamp.Unix() {
				pod = p
			}
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return false, errors.Errorf(
				"cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
		}
		if pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return "", errors.Errorf(
			"cannot exec into a container in a pod until that pod is running; current phase is %s", pod.Status.Phase)
	}

	var stdout, stderr bytes.Buffer

	req := clientSet.CoreV1().RESTClient().Post().
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

	exec, err := remotecommand.NewSPDYExecutor(a.hostConfig, "POST", req.URL())
	if err != nil {
		return "", err
	}
	if err := exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	}); err != nil {
		return "", err
	}
	kubeConfig, err := clientcmd.Load(stdout.Bytes())
	if err != nil {
		return "", errors.WithStack(err)
	}

	svc, err := clientSet.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return "", errors.WithStack(err)
	}

	var addr, port string
	if svc.Spec.Type == corev1.ServiceTypeNodePort {
		nodes, err := clientSet.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "", errors.WithStack(err)
		}
		if len(nodes.Items) == 0 {
			return "", errors.New("can not find nodes")
		}

		addrs := nodes.Items[0].Status.Addresses

		for _, a := range addrs {
			if a.Type == corev1.NodeExternalIP {
				addr = a.Address
				break
			}
		}
	}
	if addr != "" {
		ports := svc.Spec.Ports
		for _, p := range ports {
			if p.TargetPort.String() == "8443" {
				port = strconv.Itoa(int(p.NodePort))
			}
		}
	}

	newName := "vcluster-" + name
	newCluster := api.NewCluster()
	newCtx := api.NewContext()
	if addr != "" && port != "" {
		for _, cluster := range kubeConfig.Clusters {
			cluster.Server = fmt.Sprintf("https://%s:%s", addr, port)
			cluster.InsecureSkipTLSVerify = true
			cluster.CertificateAuthorityData = nil
			newCluster = cluster
		}
	}
	for _, ctx := range kubeConfig.Contexts {
		ctx.Cluster = newName
		newCtx = ctx
	}
	kubeConfig.CurrentContext = newName
	kubeConfig.Contexts = map[string]*api.Context{
		newName: newCtx,
	}
	kubeConfig.Clusters = map[string]*api.Cluster{
		newName: newCluster,
	}

	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(out), nil
}

func NewAuthConfig(hostConfig *rest.Config) AuthConfig {
	return &authConfig{
		hostConfig: hostConfig,
	}
}
