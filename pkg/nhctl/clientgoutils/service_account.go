/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) CreateServiceAccount(name string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	sa.Name = name
	sa.Labels = c.labels
	sa, err := c.ClientSet.CoreV1().ServiceAccounts(c.namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
	return sa, errors.Wrap(err, "")
}

func (c *ClientGoUtils) CreateServiceAccountINE(name string) (*corev1.ServiceAccount, error) {
	if sa, err := c.CreateServiceAccount(name); err != nil && !k8serrors.IsAlreadyExists(err) {
		return sa, err
	} else {
		return sa, nil
	}
}

func (c *ClientGoUtils) GetServiceAccount(name string) (*corev1.ServiceAccount, error) {
	return c.ClientSet.CoreV1().ServiceAccounts(c.namespace).Get(context.TODO(), name, metav1.GetOptions{})
}
