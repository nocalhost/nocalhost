/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultCIDR      = "10.96.0.0/12"
	defaultCIDRIndex = "The range of valid IPs is"
)

func getCIDR(config *rest.Config, namespace string) string {
	clientSet, err := kubernetes.NewForConfig(config)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "foo-svc-",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			ClusterIP: "8.8.8.8",
		},
	}
	_, err = clientSet.CoreV1().Services(namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	if err == nil {
		return defaultCIDR
	}

	errstr := err.Error()
	idx := strings.LastIndex(errstr, defaultCIDRIndex)
	if idx == -1 {
		return defaultCIDR
	}
	return strings.TrimSpace(err.Error()[idx+len(defaultCIDRIndex):])
}
