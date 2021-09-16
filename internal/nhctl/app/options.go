/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

type FileSyncOptions struct {
	//RunAsDaemon    bool
	SyncDouble     bool
	SyncedPattern  []string
	IgnoredPattern []string
	Override       bool
	Container      string // container name of pod to sync
	Resume         bool
	Stop           bool
}

type SyncStatusOptions struct {
	Override    bool
	WaitForSync bool
	Watch       bool
	Timeout     int64
}

type SyncStatusDirOptions struct {
	Override    bool
	WaitForSync bool
	Timeout     int64
}
