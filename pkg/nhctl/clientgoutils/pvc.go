/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// quantityStr: 10Gi, 10Mi ...
// storageClassName: nil to use default storageClassName
func (c *ClientGoUtils) CreatePVC(
	name string, labels map[string]string, annotations map[string]string, quantityStr string, storageClassName *string,
) (*v1.PersistentVolumeClaim, error) {
	q, err := resource.ParseQuantity(quantityStr)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	var persistentVolumeClaim = &v1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Selector:    nil,
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{v1.ResourceStorage: q},
			},
			StorageClassName: storageClassName,
		},
	}
	persistentVolumeClaim.Name = name
	persistentVolumeClaim.Labels = labels
	persistentVolumeClaim.Annotations = annotations

	return c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).Create(
		c.ctx, persistentVolumeClaim, metav1.CreateOptions{},
	)
}

func (c *ClientGoUtils) DeletePVC(name string) error {
	return errors.Wrap(c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).Delete(
		c.ctx, name, metav1.DeleteOptions{}), "")
}

func (c *ClientGoUtils) ListPvcs() ([]v1.PersistentVolumeClaim, error) {
	list, err := c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return list.Items, nil
}

func (c *ClientGoUtils) GetPvcByLabels(labels map[string]string) ([]v1.PersistentVolumeClaim, error) {
	var labelSelector string
	if len(labels) > 0 {
		for key, val := range labels {
			labelSelector = fmt.Sprintf("%s,%s=%s", labelSelector, key, val)
		}
	}
	labelSelector = strings.TrimPrefix(labelSelector, ",")

	list, err := c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).List(
		c.ctx, metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return list.Items, nil
}

func (c *ClientGoUtils) GetPvcByName(name string) (*v1.PersistentVolumeClaim, error) {
	pvc, err := c.ClientSet.CoreV1().PersistentVolumeClaims(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return pvc, nil
}
