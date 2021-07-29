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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
)

type ClientGoUtilClient struct {
	ClientInner     *clientgoutils.ClientGoUtils
	KubeconfigBytes []byte
}

func (cso *ClientGoUtilClient) ExecHook(appName, ns, manifests string) error {
	return cso.ClientInner.ApplyAndWaitFor(
		manifests, true,
		clientgoutils.StandardNocalhostMetas(appName, ns).
			SetDoApply(true).
			SetBeforeApply(nil),
	)
}
func (cso *ClientGoUtilClient) CleanManifest(manifests string) {
	resource := clientgoutils.NewResourceFromStr(manifests)

	//goland:noinspection GoNilness
	infos, err := resource.GetResourceInfo(cso.ClientInner, true)
	utils.ShouldI(err, "Error while loading the manifest able to be deleted: "+manifests)

	for _, info := range infos {
		utils.ShouldI(clientgoutils.DeleteResourceInfo(info), "Failed to delete resource "+info.Name)
	}
}
func (cso *ClientGoUtilClient) Create(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Create(
		cso.ClientInner.GetContext(), secret, metav1.CreateOptions{},
	)
}
func (cso *ClientGoUtilClient) Update(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Update(
		cso.ClientInner.GetContext(), secret, metav1.UpdateOptions{},
	)
}
func (cso *ClientGoUtilClient) Delete(ns, name string) error {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Delete(
		cso.ClientInner.GetContext(), name, metav1.DeleteOptions{},
	)
}
func (cso *ClientGoUtilClient) GetKubeconfigBytes() []byte {
	return cso.KubeconfigBytes
}
