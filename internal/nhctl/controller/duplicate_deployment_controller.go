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

const (
	IdentifierKey         = "identifier"
	OriginWorkloadNameKey = "origin-workload-name"
	OriginWorkloadTypeKey = "origin-workload-type"
)

type DuplicateDeploymentController struct {
	*Controller
}

func (d *DuplicateDeploymentController) GetNocalhostDevContainerPod() (string, error) {
	return d.GetDuplicateModeDevContainerPodName()
}

// ReplaceImage Create a duplicate deployment instead of replacing image
func (d *DuplicateDeploymentController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return d.ReplaceDuplicateModeImage(ctx, ops)
}

func (d *DuplicateDeploymentController) RollBack(reset bool) error {
	return d.DuplicateModeRollBack()
}

func (d *DuplicateDeploymentController) GetPodList() ([]corev1.Pod, error) {
	return d.GetDuplicateModePodList()
}
