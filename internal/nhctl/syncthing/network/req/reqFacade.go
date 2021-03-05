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

	// first find if is syncing
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
			Tips:      "Every thing is up to date.",
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
