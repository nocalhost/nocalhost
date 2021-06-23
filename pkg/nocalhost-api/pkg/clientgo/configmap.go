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
