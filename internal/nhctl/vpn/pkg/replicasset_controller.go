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

type ReplicasController struct {
	factory   cmdutil.Factory
	clientset *kubernetes.Clientset
	namespace string
	name      string
}

func NewReplicasController(factory cmdutil.Factory, clientset *kubernetes.Clientset, namespace, name string) *ReplicasController {
	return &ReplicasController{
		factory:   factory,
		clientset: clientset,
		namespace: namespace,
		name:      name,
	}
}

func (c *ReplicasController) ScaleToZero() (map[string]string, []v1.ContainerPort, string, error) {
	replicaSet, err := c.clientset.AppsV1().ReplicaSets(c.namespace).Get(context.TODO(), c.name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, "", err
	}
	if err = util.UpdateReplicasScale(c.clientset, c.namespace, util.ResourceTupleWithScale{
		Resource: c.getResource(),
		Name:     c.name,
		Scale:    0,
	}); err != nil {
		return nil, nil, "", err
	}
	formatInt := strconv.FormatInt(int64(*replicaSet.Spec.Replicas), 10)
	return replicaSet.Spec.Template.Labels, replicaSet.Spec.Template.Spec.Containers[0].Ports, formatInt, nil
}

func (c *ReplicasController) Cancel() error {
	return c.Reset()
}

func (c *ReplicasController) getResource() string {
	return "replicasets"
}

func (c *ReplicasController) Reset() error {
	get, err := c.clientset.CoreV1().
		Pods(c.namespace).
		Get(context.TODO(), ToInboundPodName(c.getResource(), c.name), metav1.GetOptions{})
	if err != nil {
		return err
	}
	if o := get.GetAnnotations()[util.OriginData]; len(o) != 0 {
		if n, err := strconv.Atoi(o); err == nil {
			if err = util.UpdateReplicasScale(c.clientset, c.namespace, util.ResourceTupleWithScale{
				Resource: c.getResource(),
				Name:     c.name,
				Scale:    n,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
