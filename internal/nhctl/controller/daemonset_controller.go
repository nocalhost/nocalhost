/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/model"
)

type DaemonSetController struct {
	*Controller
}

func (d *DaemonSetController) GetNocalhostDevContainerPod() (string, error) {
	checkPodsList, err := d.ListPodOfGeneratedDeployment()
	if err != nil {
		return "", err
	}
	return findDevPodName(checkPodsList)
}

// ReplaceImage For DaemonSet, we don't replace the DaemonSet' image
// but create a deployment with dev container instead
func (d *DaemonSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return d.PatchDevModeManifest(ctx, ops)
}

func (d *DaemonSetController) RollBack(reset bool) error {
	return d.RollbackFromAnnotation()
}

// GetPodList
// In DevMode, return pod list of generated Deployment.
// Otherwise, return pod list of DaemonSet
func (d *DaemonSetController) GetPodList() ([]corev1.Pod, error) {
	if d.IsInReplaceDevMode() {
		return d.ListPodOfGeneratedDeployment()
	}
	return d.Controller.GetPodList()
}
