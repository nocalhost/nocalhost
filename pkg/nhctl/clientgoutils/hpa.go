/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"github.com/pkg/errors"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *ClientGoUtils) ListHPA() ([]autoscalingv1.HorizontalPodAutoscaler, error) {
	ops := metav1.ListOptions{}
	if len(c.labels) > 0 {
		ops.LabelSelector = labels.Set(c.labels).String()
	}
	hpaList, err := c.ClientSet.AutoscalingV1().HorizontalPodAutoscalers(c.namespace).List(c.ctx, ops)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	result := make([]autoscalingv1.HorizontalPodAutoscaler, 0)
	for _, hpa := range hpaList.Items {
		if hpa.DeletionTimestamp == nil {
			result = append(result, hpa)
		}
	}
	return result, nil
}
