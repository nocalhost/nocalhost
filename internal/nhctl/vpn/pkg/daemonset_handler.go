/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
)

type DaemonSetHandler struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewDaemonSetHandler(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *DaemonSetHandler {
	return &DaemonSetHandler{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

// ScaleToZero update DaemonSet spec nodeName to a not exist one, then create a new pod
func (d *DaemonSetHandler) ScaleToZero() (map[string]string, map[string]string, []v1.ContainerPort, string, error) {
	daemonSet, err := d.clientset.AppsV1().DaemonSets(d.namespace).Get(context.TODO(), d.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, nil, "", err
	}
	s := daemonSet.Spec.Template.Spec.NodeName
	daemonSet.Spec.Template.Spec.NodeName = util.GetMacAddress().String()
	update, err := d.clientset.AppsV1().DaemonSets(d.namespace).Update(context.TODO(), daemonSet, metav1.UpdateOptions{})
	return update.Spec.Template.GetLabels(), update.GetAnnotations(), update.Spec.Template.Spec.Containers[0].Ports, s, nil
}

func (d *DaemonSetHandler) getResource() string {
	return "daemonsets"
}

func (d DaemonSetHandler) ToInboundPodName() string {
	return fmt.Sprintf("%s-%s-shadow", d.getResource(), d.name)
}

func (d *DaemonSetHandler) Reset() error {
	pod, err := d.clientset.CoreV1().
		Pods(d.namespace).
		Get(context.TODO(), d.ToInboundPodName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if nodeName, ok := pod.GetAnnotations()[util.OriginData]; ok {
		daemonSet, err := d.clientset.AppsV1().DaemonSets(d.namespace).Get(context.TODO(), d.name, metav1.GetOptions{})
		if err == nil {
			daemonSet.Spec.Template.Spec.NodeName = nodeName
			_, _ = d.clientset.AppsV1().DaemonSets(d.namespace).Update(context.TODO(), daemonSet, metav1.UpdateOptions{})
		}
	}
	return nil
}
