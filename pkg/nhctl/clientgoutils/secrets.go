/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) CreateSecret(secret *corev1.Secret, options metav1.CreateOptions) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(c.namespace).Create(c.ctx, secret, options)
}

func (c *ClientGoUtils) UpdateSecret(secret *corev1.Secret, options metav1.UpdateOptions) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(c.namespace).Update(c.ctx, secret, options)
}

func (c *ClientGoUtils) GetSecret(name string) (*corev1.Secret, error) {
	return c.ClientSet.CoreV1().Secrets(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
}

func (c *ClientGoUtils) DeleteSecret(name string) error {
	return c.ClientSet.CoreV1().Secrets(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{})
}