/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"nocalhost/internal/nhctl/model"
)

type DefaultController struct {
	*Controller
}

func (j *DefaultController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return j.Controller.ReplaceImage(ctx, ops)
}

func (j *DefaultController) RollBack(reset bool) error {
	return j.Controller.RollBack(reset)
}
