/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/const"
)

// Get all PersistVolumeClaims created by this application
func (a *Application) GetAllPVCs() ([]v1.PersistentVolumeClaim, error) {
	return a.client.GetPvcByLabels(map[string]string{_const.AppLabel: a.Name})
}
