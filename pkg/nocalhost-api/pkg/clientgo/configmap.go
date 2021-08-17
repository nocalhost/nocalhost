/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgo

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *GoClient) GetConfigMap(namespace, name string) (*corev1.ConfigMap, error) {
	resource, err := c.client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return resource, errors.WithStack(err)
}

// GetConfigMapList get a list of configmaps.
func (c *GoClient) GetConfigMapList(namespace string) (*corev1.ConfigMapList, error) {
	resources, err := c.client.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	return resources, errors.WithStack(err)
}

func (c *GoClient) CreateConfigMap(namespace string, configmap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	resource, err := c.client.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configmap, metav1.CreateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) UpdateConfigMap(namespace string, configmap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	resource, err := c.client.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) DeleteConfigMap(namespace string, configmap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	resource, err := c.client.CoreV1().ConfigMaps(namespace).Update(context.TODO(), configmap, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}
