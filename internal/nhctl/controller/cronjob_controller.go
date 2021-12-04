/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/model"
)

type CronJobController struct {
	*Controller
}

const (
	cronjobGeneratedJobPrefix = "cronjob-generated-job-"
	//cronjobSuspendAnnotation  = "nocalhost.dev.cronjob.suspend"
)

func (j *CronJobController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := j.GetPodList()
	if err != nil {
		return "", err
	}
	return findDevPodName(checkPodsList)
}

func (j *CronJobController) getGeneratedJobName() string {
	return fmt.Sprintf("%s%s", cronjobGeneratedJobPrefix, j.GetName())
}

// ReplaceImage For Job, we can't replace the Job' image
// but create a Deployment with dev container instead
func (j *CronJobController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return j.PatchDevModeManifest(ctx, ops)
}

func (j *CronJobController) RollBack(reset bool) error {
	return j.RollbackFromAnnotation()
}

// GetPodList
// In DevMode, return pod list of generated Job.
// Otherwise, return pod list of original Deployment
func (j *CronJobController) GetPodList() ([]corev1.Pod, error) {
	if j.IsInReplaceDevMode() {
		return j.ListPodOfGeneratedDeployment()
	}
	return j.Controller.GetPodList()
}
