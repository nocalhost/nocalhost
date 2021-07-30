/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
