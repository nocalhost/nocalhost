/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package vcluster

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"

	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
)

func GetKubeConfig(clusterKubeConfig, name, namespace string) (string, error) {
	goClient, err := clientgo.NewAdminGoClient([]byte(clusterKubeConfig))
	if err != nil {
		return "", err
	}

	options := metav1.ListOptions{LabelSelector: fmt.Sprintf("app=vcluster,release=%s", name)}
	pods, err := goClient.ListPodsByOptions(namespace, options)
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", errors.New("can not find vcluster pod")
	}
	sort.Slice(pods.Items, func(i, j int) bool {
		return pods.Items[i].CreationTimestamp.Unix() > pods.Items[j].CreationTimestamp.Unix()
	})
	pod := pods.Items[0]
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return "", errors.Errorf(
			"cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	var stdout, stderr bytes.Buffer

	goClient.GetClientSet()

	req := goClient.GetClientSet().CoreV1().RESTClient().Post().
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

	exec, err := remotecommand.NewSPDYExecutor(goClient.GetRestConfig(), "POST", req.URL())
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

	svc, err := goClient.GetService(namespace, name)
	if err != nil {
		return "", errors.WithStack(err)
	}

	var addr, port string
	if svc.Spec.Type == corev1.ServiceTypeNodePort {
		nodes, err := goClient.GetClusterNode()
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

	if addr != "" && port != "" {
		for cluster := range kubeConfig.Clusters {
			kubeConfig.Clusters[cluster].Server = fmt.Sprintf("https://%s:%s", addr, port)
			kubeConfig.Clusters[cluster].InsecureSkipTLSVerify = true
			kubeConfig.Clusters[cluster].CertificateAuthorityData = nil
		}
	}

	out, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return string(out), nil
}
