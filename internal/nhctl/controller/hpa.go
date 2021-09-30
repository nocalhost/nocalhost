/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import "k8s.io/apimachinery/pkg/fields"

func (c *Controller) GetHpa() error {
	typeMeta, err := c.GetTypeMeta()
	if err != nil {
		return err
	}

	c.Client.FieldSelector(
		fields.AndSelectors(
			fields.OneTermEqualSelector("spec.scaleTargetRef.apiVersion", typeMeta.APIVersion),
			fields.OneTermEqualSelector("spec.scaleTargetRef.kind", typeMeta.Kind),
			fields.OneTermEqualSelector("spec.scaleTargetRef.name", c.Name),
		).String(),
	)

}
