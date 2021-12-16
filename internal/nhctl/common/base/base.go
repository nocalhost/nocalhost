/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package base

type DevModeAction struct {
	ScaleAction     []PatchItem
	PodTemplatePath string
	Create          bool
}

type CrdDevModeAction struct {
	CrdType       string         `json:"crd_type" yaml:"crd_type"`
	DevModeAction *DevModeAction `json:"dev_mode_action" yaml:"dev_mode_action"`
}

type ConfigFile struct {
	NhEsUrl           string             `json:"nh_es_url" yaml:"nh_es_url"`
	CrdDevModeActions []CrdDevModeAction `json:"crd_dev_mode_actions" yaml:"crd_dev_mode_actions"`
}

type PatchItem struct {
	Patch string `json:"patch" yaml:"patch"`
	Type  string `json:"type,omitempty" yaml:"type,omitempty"`
}
