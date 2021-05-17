/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
		return disconnectedTemplate
	}

	status, err := p.FolderStatus()
	if err != nil {
		return &SyncthingStatus{
			Status: Error,
			Msg:    "Disconnected",
			Tips:   fmt.Sprintf("%v", err),
		}
	}

	if status.hasError() {
		return &SyncthingStatus{
			Status: Error,
			Msg:    "Error",
			Tips:   fmt.Sprintf("%v", err),
		}
	}

	completion, err := p.Completion()
	if err != nil {
		return &SyncthingStatus{
			Status:    Error,
			Msg:       "Error",
			Tips:      fmt.Sprintf("%v", err),
			OutOfSync: status.OutOfSync(),
		}
	}

	// first check need synced force
	if completion.NeedOverrideForce() {
		return &SyncthingStatus{
			Status:    OutOfSync,
			Msg:       status.OutOfSyncLog(),
			Tips:      status.OutOfSyncTips(),
			OutOfSync: status.OutOfSync(),
		}
	}

	// then find if is syncing
	if !completion.isComplete() {
		return &SyncthingStatus{
			Status:    Syncing,
			Msg:       completion.UploadPct(),
			Tips:      status.StateChangedLog(),
			OutOfSync: status.OutOfSync(),
		}
	}

	// then idle
	if status.isIdle() {
		return &SyncthingStatus{
			Status:    Idle,
			Msg:       status.StateChangedLog(),
			Tips:      status.IdleTips(),
			OutOfSync: status.OutOfSync(),
		}
	}

	// else means the local folder is in scanning
	return &SyncthingStatus{
		Status:    Scanning,
		Msg:       "Scanning local changed...",
		OutOfSync: status.OutOfSync(),
	}
}

// the nhctl sync --status result
type SyncthingStatus struct {
	Status    StatusEnum `json:"status"`
	Msg       string     `json:"msg"`
	Tips      string     `json:"tips,omitempty"`
	OutOfSync string     `json:"outOfSync,omitempty"`
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

var NotInDevModeTemplate = &SyncthingStatus{
	Status: End,
	Msg:    "Not in DevMode",
	Tips:   Identifier + "File sync does not working due to the devMode is not enabled.",
}

var FileSyncNotRunningTemplate = &SyncthingStatus{
	Status: End,
	Msg:    "File sync is not running!",
	Tips:   Identifier + "File sync does not working, please try to reenter devMode!",
}

var disconnectedTemplate = &SyncthingStatus{
	Status: Disconnected,
	Msg:    "Disconnected from sidecar",
	Tips:   Identifier + "Please check your network connection and ensure the port-forward from sidecar is valid.",
}
