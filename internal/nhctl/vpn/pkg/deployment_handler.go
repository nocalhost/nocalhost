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
	"strconv"
)

type DeploymentHandler struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewDeploymentHandler(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *DeploymentHandler {
	return &DeploymentHandler{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (d *DeploymentHandler) ScaleToZero() (map[string]string, map[string]string, []v1.ContainerPort, string, error) {
	scale, err := d.clientset.AppsV1().Deployments(d.namespace).GetScale(context.TODO(), d.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, nil, "", err
	}
	if err = util.UpdateReplicasScale(d.clientset, d.namespace, util.ResourceTupleWithScale{
		Resource: d.getResource(),
		Name:     d.name,
		Scale:    0,
	}); err != nil {
		return nil, nil, nil, "", err
	}
	get, err := d.clientset.AppsV1().Deployments(d.namespace).Get(context.TODO(), d.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, nil, "", err
	}
	formatInt := strconv.FormatInt(int64(scale.Spec.Replicas), 10)
	return get.Spec.Template.GetLabels(), get.GetAnnotations(), get.Spec.Template.Spec.Containers[0].Ports, formatInt, nil
}

func (d *DeploymentHandler) getResource() string {
	return "deployments"
}

func (d DeploymentHandler) ToInboundPodName() string {
	return fmt.Sprintf("%s-%s-shadow", d.getResource(), d.name)
}

func (d *DeploymentHandler) Reset() error {
	pod, err := d.clientset.CoreV1().
		Pods(d.namespace).
		Get(context.TODO(), d.ToInboundPodName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if o := pod.GetAnnotations()[util.OriginData]; len(o) != 0 {
		if n, err := strconv.Atoi(o); err == nil {
			if err = util.UpdateReplicasScale(d.clientset, d.namespace, util.ResourceTupleWithScale{
				Resource: d.getResource(),
				Name:     d.name,
				Scale:    n,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
