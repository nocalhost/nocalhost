/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package base

type DevModeAction struct {
	ScalePatches    []PatchItem `json:"scalePatches" yaml:"scalePatches"`
	PodTemplatePath string      `json:"podTemplatePath" yaml:"podTemplatePath"`
	Create          bool        `json:"create" yaml:"create"`
}

type CrdDevModeAction struct {
	CrdType       string         `json:"crdType" yaml:"crdType"`
	DevModeAction *DevModeAction `json:"devModeAction" yaml:"devModeAction"`
}

type ConfigFile struct {
	NhEsUrl           string             `json:"nhEsUrl" yaml:"nhEsUrl"`
	CrdDevModeActions []CrdDevModeAction `json:"crdDevModeActions" yaml:"crdDevModeActions"`
}

type PatchItem struct {
	Patch string `json:"patch" yaml:"patch"`
	Type  string `json:"type,omitempty" yaml:"type,omitempty"`
}
