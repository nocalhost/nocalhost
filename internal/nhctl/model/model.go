/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

type NocalHostResource struct {
	NameSpace   string
	Nid         string
	Application string
	Service     string
	ServiceType string
	PodName     string
}

type DevStartOptions struct {
	SideCarImage string
	DevImage     string
	Container    string

	// for debug
	SyncthingVersion string

	// Now it's only use to specify the `root dir` user want to sync
	LocalSyncDir  []string
	StorageClass  string
	PriorityClass string

	NoTerminal  bool
	NoSyncthing bool

	DevModeType string
	MeshHeader  map[string]string
}
