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

func (c *ClientGoUtils) UpdateHPA(hpa *autoscalingv1.HorizontalPodAutoscaler) (*autoscalingv1.HorizontalPodAutoscaler, error) {
	hpa2, err := c.ClientSet.AutoscalingV1().HorizontalPodAutoscalers(c.namespace).
		Update(c.ctx, hpa, metav1.UpdateOptions{})
	return hpa2, errors.Wrap(err, "")
}

func (c *ClientGoUtils) ListHPA() ([]autoscalingv1.HorizontalPodAutoscaler, error) {
	ops := metav1.ListOptions{}
	if len(c.labels) > 0 {
		ops.LabelSelector = labels.Set(c.labels).String()
	}
	if len(c.fieldSelector) > 0 {
		ops.FieldSelector = c.fieldSelector
	}
	hpaList, err := c.ClientSet.AutoscalingV1().HorizontalPodAutoscalers(c.namespace).List(c.ctx, ops)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	result := make([]autoscalingv1.HorizontalPodAutoscaler, 0)
	if !c.includeDeletedResources {
		for _, hpa := range hpaList.Items {
			if hpa.DeletionTimestamp == nil {
				result = append(result, hpa)
			}
		}
	} else {
		result = hpaList.Items
	}
	return result, nil
}
