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

type DuplicateStatefulSetController struct {
	*Controller
}

func (s *DuplicateStatefulSetController) GetNocalhostDevContainerPod() (string, error) {
	return s.GetDuplicateModeDevContainerPodName()
}

func (s *DuplicateStatefulSetController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return s.ReplaceDuplicateModeImage(ctx, ops)
}

func (s *DuplicateStatefulSetController) RollBack(reset bool) error {
	return s.DuplicateModeRollBack()
}

func (s *DuplicateStatefulSetController) GetPodList() ([]corev1.Pod, error) {
	return s.GetDuplicateModePodList()
}
