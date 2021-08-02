/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

// Test Client GoUtils Get Resources By Rest Client
func TestClientGoUtilsGRBRC(t *testing.T) {
	client, err := NewClientGoUtils("", "")
	Must(err)
	result := &corev1.PodList{}
	Must(client.GetResourcesByRestClient(&corev1.SchemeGroupVersion, ResourcePods, result))
	for _, item := range result.Items {
		fmt.Println(item.Name)
	}
}
