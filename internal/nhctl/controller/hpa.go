/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/fields"
)

func (c *Controller) ListHPA() ([]autoscalingv1.HorizontalPodAutoscaler, error) {
	typeMeta, err := c.GetTypeMeta()
	if err != nil {
		return nil, err
	}

	return c.Client.FieldSelector(
		fields.AndSelectors(
			fields.OneTermEqualSelector("spec.scaleTargetRef.apiVersion", typeMeta.APIVersion),
			fields.OneTermEqualSelector("spec.scaleTargetRef.kind", typeMeta.Kind),
			fields.OneTermEqualSelector("spec.scaleTargetRef.name", c.Name),
		).String(),
	).ListHPA()
}
