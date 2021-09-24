/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pod_controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/model"
)

type CommonController interface {
	IsInDevMode() bool
	GetName() string // Controller name
}

type PodController interface {
	CommonController
	ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error
	GetNocalhostDevContainerPod() (string, error) // Dev container not found, return err
	//Name() string                                 // Controller name
	RollBack(reset bool) error
	GetPodList() ([]corev1.Pod, error) // Find pods to port-forward and enter terminal
}
