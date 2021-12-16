/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"nocalhost/internal/nhctl/vpn/util"
	"strconv"
)

type DeploymentController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewDeploymentController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *DeploymentController {
	return &DeploymentController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (d *DeploymentController) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	scale, err := d.clientset.AppsV1().Deployments(d.namespace).GetScale(context.TODO(), d.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, "", err
	}
	if err = util.UpdateReplicasScale(d.clientset, d.namespace, util.ResourceTupleWithScale{
		Resource: d.getResource(),
		Name:     d.name,
		Scale:    0,
	}); err != nil {
		return nil, nil, "", err
	}
	get, err := d.clientset.AppsV1().Deployments(d.namespace).Get(context.TODO(), d.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, "", err
	}
	formatInt := strconv.FormatInt(int64(scale.Spec.Replicas), 10)
	return get.Spec.Template.GetLabels(), get.Spec.Template.Spec.Containers[0].Ports, formatInt, nil
}

func (d *DeploymentController) Cancel() error {
	return d.Reset()
}

func (d *DeploymentController) getResource() string {
	return "deployments"
}

func (d *DeploymentController) Reset() error {
	pod, err := d.clientset.CoreV1().
		Pods(d.namespace).
		Get(context.TODO(), ToInboundPodName(d.getResource(), d.name), metav1.GetOptions{})
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
