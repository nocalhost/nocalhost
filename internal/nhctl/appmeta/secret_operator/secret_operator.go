/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
func (cso *ClientGoUtilClient) ReObtainSecret(ns string, secret *corev1.Secret) (*corev1.Secret, error) {
	return cso.ClientInner.ClientSet.CoreV1().Secrets(ns).Get(
		cso.ClientInner.GetContext(), secret.Name, metav1.GetOptions{},
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
