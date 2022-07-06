/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package req

import (
	"fmt"
)

func (p *SyncthingHttpClient) GetSyncthingStatus() *SyncthingStatus {
	status := p.getSyncthingStatus()

	if status.Tips != "" {
		status.Tips = Identifier + status.Tips
	}

	if status.OutOfSync != "" {
		status.OutOfSync = Identifier + status.OutOfSync
	}

	return status
}

func (p *SyncthingHttpClient) getSyncthingStatus() *SyncthingStatus {

	// The SyncthingStatus is consist of three parts
	// 1) Check the connection
	// 2) Check the folder
	// 3) Check the device

	connections, err := p.SystemConnections()
	if err != nil || !connections {
		return disconnectedTemplateGen(p.guiHost)
	}

	status, err := p.FolderStatus()
	if err != nil {
		return &SyncthingStatus{
			Status: Error,
			Msg:    "Disconnected",
			Tips:   fmt.Sprintf("%v", err),
			Gui:    p.guiHost,
		}
	}

	if status.hasError() {
		return &SyncthingStatus{
			Status: Error,
			Msg:    "Error",
			Tips:   fmt.Sprintf("%v", err),
			Gui:    p.guiHost,
		}
	}

	completion, err := p.Completion()
	if err != nil {
		return &SyncthingStatus{
			Status:    Error,
			Msg:       "Error",
			Tips:      fmt.Sprintf("%v", err),
			OutOfSync: status.OutOfSync(),
			Gui:       p.guiHost,
		}
	}

	// first check need synced force
	if completion.NeedOverrideForce() {
		return &SyncthingStatus{
			Status:    OutOfSync,
			Msg:       status.OutOfSyncLog(),
			Tips:      status.OutOfSyncTips(),
			OutOfSync: status.OutOfSync(),
			Gui:       p.guiHost,
		}
	}

	// then find if is syncing
	if !completion.isComplete() {
		return &SyncthingStatus{
			Status:    Syncing,
			Msg:       completion.UploadPct(),
			Tips:      status.StateChangedLog(),
			OutOfSync: status.OutOfSync(),
			Gui:       p.guiHost,
		}
	}

	// then idle
	if status.isIdle() {
		return &SyncthingStatus{
			Status:    Idle,
			Msg:       status.StateChangedLog(),
			Tips:      status.IdleTips(),
			OutOfSync: status.OutOfSync(),
			Gui:       p.guiHost,
		}
	}

	// else means the local folder is in scanning
	return &SyncthingStatus{
		Status:    Scanning,
		Msg:       "Scanning local changed...",
		OutOfSync: status.OutOfSync(),
		Gui:       p.guiHost,
	}
}

// the nhctl sync --status result
type SyncthingStatus struct {
	Status    StatusEnum `yaml:"status" json:"status"`
	Msg       string     `json:"msg"`
	Tips      string     `json:"tips,omitempty"`
	OutOfSync string     `json:"outOfSync,omitempty"`
	Gui       string     `json:"gui,omitempty"`
}

type StatusEnum string

// Use strings as keys to make printout and serialization of the locations map
// more meaningful.
const (
	Disconnected StatusEnum = "disconnected"
	OutOfSync    StatusEnum = "outOfSync"
	Scanning     StatusEnum = "scanning"
	Syncing      StatusEnum = "syncing"
	Error        StatusEnum = "error"
	Idle         StatusEnum = "idle"
	End          StatusEnum = "end"

	Identifier = "(Nocalhost): "
)

var WelcomeTemplate = &SyncthingStatus{
	Status: End,
	Msg:    "Welcome to Nocalhost",
	Tips:   Identifier + "There is no service associated with the current directory",
}

var AppNotInstalledTemplate = &SyncthingStatus{
	Status: End,
	Msg:    "Application not installed",
	Tips:   Identifier + "Current Application is not installed, please check your application is installed correctly.",
}

var NotInDevModeTemplate = &SyncthingStatus{
	Status: End,
	Msg:    "Not in DevMode",
	Tips:   Identifier + "File sync does not working due to the devMode is not enabled.",
}

var DevModeStarting = &SyncthingStatus{
	Status: End,
	Msg:    "DevMode Starting...",
	Tips:   Identifier + "File sync in preparation.",
}

var NotProcessor = &SyncthingStatus{
	Status: End,
	Msg:    "Other device is developing",
	Tips: Identifier + "File Sync is hold by other device, if you want to take over the file sync, " +
		"you should end the dev mode and re enter again.",
}

var NotSyncthingProcessFound = &SyncthingStatus{
	Status: Disconnected,
	Msg:    "No syncthing process found",
	Tips:   Identifier + "No syncthing process found, please restart it.",
}

func disconnectedTemplateGen(guiHost string) *SyncthingStatus {
	return &SyncthingStatus{
		Status: Disconnected,
		Msg:    "Disconnected from sidecar",
		Tips:   Identifier + "Please check your network connection and ensure the port-forward from sidecar is valid.",
		Gui:    guiHost,
	}
}
