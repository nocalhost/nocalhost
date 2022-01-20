/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *ClientGoUtils) DeleteConfigMapByName(name string) error {
	return errors.Wrap(c.ClientSet.CoreV1().ConfigMaps(c.namespace).Delete(c.ctx, name, metav1.DeleteOptions{}), "")
}

func (c *ClientGoUtils) ListConfigMaps() ([]v1.ConfigMap, error) {
	var result []v1.ConfigMap
	list, err := c.ClientSet.CoreV1().ConfigMaps(c.namespace).List(c.ctx, metav1.ListOptions{})
	if list != nil {
		result = list.Items
	}
	return result, errors.Wrap(err, "")
}

func (c *ClientGoUtils) UpdateConfigMaps(cm *v1.ConfigMap) (*v1.ConfigMap, error) {
	cm2, err := c.ClientSet.CoreV1().ConfigMaps(c.namespace).Update(c.ctx, cm, metav1.UpdateOptions{})
	return cm2, errors.Wrap(err, "")
}

func (c *ClientGoUtils) GetConfigMaps(name string) (*v1.ConfigMap, error) {
	result, err := c.ClientSet.CoreV1().ConfigMaps(c.namespace).Get(c.ctx, name, metav1.GetOptions{})
	return result, errors.Wrap(err, "")
}
