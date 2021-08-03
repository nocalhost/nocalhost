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

func (c *ClientGoUtils) GetConfigMaps() ([]v1.ConfigMap, error) {
	var result []v1.ConfigMap
	list, err := c.ClientSet.CoreV1().ConfigMaps(c.namespace).List(c.ctx, metav1.ListOptions{})
	if list != nil {
		result = list.Items
	}
	return result, errors.Wrap(err, "")
}
