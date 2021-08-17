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

func (c *GoClient) GetService(namespace, name string) (*corev1.Service, error) {
	resource, err := c.client.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return resource, errors.WithStack(err)
}

// GetServiceList get a list of services.
func (c *GoClient) GetServiceList(namespace string) (*corev1.ServiceList, error) {
	resources, err := c.client.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	return resources, errors.WithStack(err)
}

func (c *GoClient) CreateService(namespace string, service *corev1.Service) (*corev1.Service, error) {
	resource, err := c.client.CoreV1().Services(namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) UpdateService(namespace string, service *corev1.Service) (*corev1.Service, error) {
	resource, err := c.client.CoreV1().Services(namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) DeleteService(namespace, name string) error {
	return errors.WithStack(c.client.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{}))
}
