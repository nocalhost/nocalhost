/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/pod_controller"
	"nocalhost/internal/nhctl/profile"
)

func (c *Controller) BuildPodController() pod_controller.PodController {
	switch c.Type {
	case base.Deployment:
		if c.DevModeType == profile.DuplicateDevMode {
			return &DuplicateDevModeController{Controller: c}
		}
		return &DeploymentController{Controller: c}
	case base.StatefulSet:
		if c.DevModeType == profile.DuplicateDevMode {
			return &DuplicateDevModeController{Controller: c}
		}
		return &DefaultController{Controller: c}
	case base.DaemonSet:
		if c.DevModeType == profile.DuplicateDevMode {
			return &DuplicateDevModeController{Controller: c}
		}
		return &DefaultController{Controller: c}
	case base.Job, base.CronJob:
		return &DefaultController{Controller: c}
	case base.Pod:
		if c.DevModeType == profile.DuplicateDevMode {
			return &DuplicateRawPodController{Controller: c}
		}
		return &RawPodController{Controller: c}
	}
	if c.DevModeType == profile.DuplicateDevMode {
		return &DuplicateDevModeController{c}
	}
	return &DefaultController{Controller: c}
}
