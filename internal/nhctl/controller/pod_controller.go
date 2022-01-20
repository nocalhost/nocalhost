/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/pod_controller"
)

func (c *Controller) BuildPodController() pod_controller.PodController {
	if c.Type == base.Pod {
		if c.DevModeType.IsDuplicateDevMode() {
			return &DuplicateRawPodController{Controller: c}
		}
		return &RawPodController{Controller: c}
	}
	return &DefaultController{Controller: c}
}
