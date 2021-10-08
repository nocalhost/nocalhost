/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

type VersionUpgradeInfo struct {
	CurrentVersion string `json:"current_version"`
	UpgradeVersion string `json:"upgrade_version"`
	HasNewVersion  bool   `json:"has_new_version"`
}
