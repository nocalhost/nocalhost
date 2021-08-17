/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgo

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *GoClient) GetDeployment(namespace, name string) (*v1.Deployment, error) {
	resource, err := c.client.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	return resource, errors.WithStack(err)
}

// GetDeploymentList get a list of deployments.
func (c *GoClient) GetDeploymentList(namespace string) (*v1.DeploymentList, error) {
	resources, err := c.client.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	return resources, errors.WithStack(err)
}

func (c *GoClient) CreateDeployment(namespace string, deployment *v1.Deployment) (*v1.Deployment, error) {
	resource, err := c.client.AppsV1().Deployments(namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) UpdateDeployment(namespace string, deployment *v1.Deployment) (*v1.Deployment, error) {
	resource, err := c.client.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) DeleteDeployment(namespace, name string) error {
	return errors.WithStack(c.client.
		AppsV1().
		Deployments(namespace).
		Delete(context.TODO(), name, metav1.DeleteOptions{}))
}
