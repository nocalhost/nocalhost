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

package secret_operator

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"nocalhost/pkg/nhctl/clientgoutils"
)

type SecretOperator interface {
	Create(ns string, secret *corev1.Secret) (*corev1.Secret, error)
	Update(ns string, secret *corev1.Secret) (*corev1.Secret, error)
	Delete(ns, name string) error
	GetKubeconfigBytes() []byte
}

type ClientGoUtilSecretOperator struct {
	ClientInner     *clientgoutils.ClientGoUtils
	KubeconfigBytes []byte
}

func (cso *ClientGoUtilSecretOperator) Create(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Create(
		cso.ClientInner.GetContext(), secret, metav1.CreateOptions{},
	)
}
func (cso *ClientGoUtilSecretOperator) Update(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Update(
		cso.ClientInner.GetContext(), secret, metav1.UpdateOptions{},
	)
}
func (cso *ClientGoUtilSecretOperator) Delete(ns, name string) error {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Delete(
		cso.ClientInner.GetContext(), name, metav1.DeleteOptions{},
	)
}
func (cso *ClientGoUtilSecretOperator) GetKubeconfigBytes() []byte {
	return cso.KubeconfigBytes
}

type ClientGoSecretOperator struct {
	ClientSet       *kubernetes.Clientset
	KubeconfigBytes []byte
}

func (cso *ClientGoSecretOperator) Create(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientSet.CoreV1().Secrets(ns).Create(
		context.TODO(), secret, metav1.CreateOptions{},
	)
}
func (cso *ClientGoSecretOperator) Update(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientSet.CoreV1().Secrets(ns).Update(
		context.TODO(), secret, metav1.UpdateOptions{},
	)
}
func (cso *ClientGoSecretOperator) Delete(ns, name string) error {
	return cso.ClientSet.CoreV1().Secrets(ns).Delete(
		context.TODO(), name, metav1.DeleteOptions{},
	)
}
func (cso *ClientGoSecretOperator) GetKubeconfigBytes() []byte {
	return cso.KubeconfigBytes
}
