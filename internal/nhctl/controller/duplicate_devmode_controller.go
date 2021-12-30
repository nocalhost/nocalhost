/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"nocalhost/internal/nhctl/model"
)

const (
	IdentifierKey         = "identifier"
	OriginWorkloadNameKey = "origin-workload-name"
	OriginWorkloadTypeKey = "origin-workload-type"
)

type DuplicateDevModeController struct {
	*Controller
}

// ReplaceImage Create a duplicate deployment instead of replacing image
func (d *DuplicateDevModeController) ReplaceImage(ctx context.Context, ops *model.DevStartOptions) error {
	return d.ReplaceDuplicateModeImage(ctx, ops)
}

func (d *DuplicateDevModeController) RollBack(reset bool) error {
	return d.DuplicateModeRollBack()
}
