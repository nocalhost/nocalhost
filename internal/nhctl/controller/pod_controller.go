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

func (c *Controller) BuildPodController(devMode profile.DevModeType) pod_controller.PodController {
	switch c.Type {
	case base.Deployment:
		if devMode == profile.DuplicateDevMode {
			return &DuplicateDeploymentController{Controller: c}
		}
		return &DeploymentController{Controller: c}
	case base.StatefulSet:
		if devMode == profile.DuplicateDevMode {
			return &DuplicateStatefulSetController{Controller: c}
		}
		return &StatefulSetController{Controller: c}
	case base.DaemonSet:
		return &DaemonSetController{Controller: c}
	case base.Job:
		return &JobController{Controller: c}
	case base.CronJob:
		return &CronJobController{Controller: c}
	case base.Pod:
		return &RawPodController{Controller: c}
	}
	return nil
}
