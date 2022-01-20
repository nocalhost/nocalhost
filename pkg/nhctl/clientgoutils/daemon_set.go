/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *ClientGoUtils) UpdateDaemonSet(ds *v1.DaemonSet) (*v1.DaemonSet, error) {
	ds, err := c.GetDaemonSetClient().Update(c.ctx, ds, metav1.UpdateOptions{})
	return ds, errors.Wrap(err, "")
}

func (c *ClientGoUtils) ListPodsByDaemonSet(name string) ([]corev1.Pod, error) {
	ss, err := c.ClientSet.AppsV1().DaemonSets(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	set := labels.Set(ss.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(
		c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pods.Items, nil
}

func (c *ClientGoUtils) ListDaemonSets() ([]v1.DaemonSet, error) {
	ops := metav1.ListOptions{}
	if len(c.labels) > 0 {
		ops.LabelSelector = labels.Set(c.labels).String()
	}
	deps, err := c.GetDaemonSetClient().List(c.ctx, ops)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return deps.Items, nil
}
