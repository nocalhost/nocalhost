/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package pod_controller

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/model"
)

type PodController interface {
	ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error
	// Container Return specified container in controller, If containerName not specified, return first container
	Container(containerName string) (*corev1.Container, error)
	Name() string // Controller name
	RollBack(reset bool) error
	GetDefaultPodNameWait(ctx context.Context) (string, error)
	GetPodList() ([]corev1.Pod, error)
}
