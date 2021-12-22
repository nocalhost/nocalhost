/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
)

func (c *Controller) ListHPA() ([]autoscalingv1.HorizontalPodAutoscaler, error) {
	typeMeta, err := c.GetTypeMeta()
	if err != nil {
		return nil, err
	}

	hpaList, err := c.Client.ListHPA()
	if err != nil {
		return nil, err
	}
	result := make([]autoscalingv1.HorizontalPodAutoscaler, 0)
	for _, hpa := range hpaList {
		hpaRef := hpa.Spec.ScaleTargetRef
		if hpaRef.APIVersion == typeMeta.APIVersion && hpaRef.Kind == typeMeta.Kind && hpaRef.Name == c.Name {
			result = append(result, hpa)
		}
	}
	return result, nil
}
