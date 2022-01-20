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

func (c *GoClient) GetSecret(namespace, name string) (*corev1.Secret, error) {
	resource, err := c.client.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return resource, errors.WithStack(err)
}

// GetSecretList get a list of secrets.
func (c *GoClient) GetSecretList(namespace string) (*corev1.SecretList, error) {
	resources, err := c.client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	return resources, errors.WithStack(err)
}

// GetSecretListByType get a list of secrets than select by type.
func (c *GoClient) GetSecretListByType(namespace string, secretType string) (*corev1.SecretList, error) {
	resources, err := c.client.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: "type=" + secretType,
	})
	return resources, errors.WithStack(err)
}

func (c *GoClient) CreateSecret(namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	resource, err := c.client.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) UpdateSecret(namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	resource, err := c.client.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) DeleteSecret(namespace string, name string) error {
	return c.client.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}
