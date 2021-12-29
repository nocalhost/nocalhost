/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ListPodsByDeployment This method can not list pods whose deployment is already deleted.
func (c *ClientGoUtils) ListPodsByDeployment(name string) (*corev1.PodList, error) {
	deployment, err := c.ClientSet.AppsV1().Deployments(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	set := labels.Set(deployment.Spec.Selector.MatchLabels)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(
		c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pods, nil
}

func (c *ClientGoUtils) ListPods() ([]corev1.Pod, error) {
	ops := c.getListOptions()
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(c.ctx, ops)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	result := make([]corev1.Pod, 0)
	if !c.includeDeletedResources {
		for _, pod := range pods.Items {
			if pod.DeletionTimestamp == nil {
				result = append(result, pod)
			}
		}
	} else {
		result = pods.Items
	}
	return result, nil
}

func (c *ClientGoUtils) ListPodsByStatefulSet(name string) (*corev1.PodList, error) {
	ss, err := c.ClientSet.AppsV1().StatefulSets(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
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
	return pods, nil
}

func (c *ClientGoUtils) ListPodsByLabels(labelMap map[string]string) ([]corev1.Pod, error) {
	set := labels.Set(labelMap)
	pods, err := c.ClientSet.CoreV1().Pods(c.namespace).List(
		c.ctx, metav1.ListOptions{LabelSelector: set.AsSelector().String()},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	result := make([]corev1.Pod, 0)
	for _, pod := range pods.Items {
		result = append(result, pod)
	}
	return result, nil
}

// DeletePodByName
// gracePeriodSeconds: The duration in seconds before the object should be deleted.
// The value zero indicates delete immediately. If this value is negative integer, the default grace period for the
// specified type will be used.
func (c *ClientGoUtils) DeletePodByName(name string, gracePeriodSeconds int64) error {
	deleteOps := metav1.DeleteOptions{}
	if gracePeriodSeconds >= 0 {
		deleteOps.GracePeriodSeconds = &gracePeriodSeconds
	}
	return errors.Wrap(c.ClientSet.CoreV1().Pods(c.namespace).Delete(c.ctx, name, deleteOps), "")
}

func (c *ClientGoUtils) CreatePod(pod *corev1.Pod) (*corev1.Pod, error) {
	pod2, err := c.ClientSet.CoreV1().Pods(c.namespace).Create(c.ctx, pod, metav1.CreateOptions{})
	return pod2, errors.Wrap(err, "")
}

func (c *ClientGoUtils) UpdatePod(pod *corev1.Pod) (*corev1.Pod, error) {
	pod2, err := c.GetPodClient().Update(c.ctx, pod, metav1.UpdateOptions{})
	return pod2, errors.Wrap(err, "")
}
