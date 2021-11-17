/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dns

import (
	"bytes"
	"context"
	miekgdns "github.com/miekg/dns"
	"github.com/pkg/errors"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"nocalhost/internal/nhctl/vpn/util"
)

func GetDNSServiceIPFromPod(clientset *kubernetes.Clientset, restclient *rest.RESTClient, config *rest.Config, podName, namespace string) (*miekgdns.ClientConfig, error) {
	var ipp []string
	if ips, err := getDNSIPFromDnsPod(clientset); err == nil && len(ips) != 0 {
		ipp = ips
	}
	if ip, err := util.Shell(clientset, restclient, config, podName, namespace, "cat /etc/resolv.conf"); err == nil {
		if resolvConf, err := miekgdns.ClientConfigFromReader(bytes.NewBufferString(ip)); err == nil {
			if len(ipp) != 0 {
				resolvConf.Servers = append(resolvConf.Servers, make([]string, len(ipp))...)
				copy(resolvConf.Servers[len(ipp):], resolvConf.Servers[:len(resolvConf.Servers)-len(ipp)])
				for i := range ipp {
					resolvConf.Servers[i] = ipp[i]
				}
			}
			return resolvConf, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func getDNSIPFromDnsPod(clientset *kubernetes.Clientset) (ips []string, err error) {
	serviceList, err := clientset.CoreV1().Pods(v1.NamespaceSystem).List(context.Background(), v1.ListOptions{
		LabelSelector: fields.OneTermEqualSelector("k8s-app", "kube-dns").String(),
	})
	if err != nil {
		return
	}
	for _, pod := range serviceList.Items {
		if pod.Status.Phase == v12.PodRunning && pod.DeletionTimestamp == nil {
			ips = append(ips, pod.Status.PodIP)
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("")
	}
	return ips, nil
}
