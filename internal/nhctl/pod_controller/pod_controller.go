/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pod_controller

import (
	"context"
	"nocalhost/internal/nhctl/model"
)

type PodController interface {
	ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error
	RollBack(reset bool) error
}
