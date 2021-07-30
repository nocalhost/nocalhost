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

func (c *GoClient) DeleteSecret(namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	resource, err := c.client.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}
