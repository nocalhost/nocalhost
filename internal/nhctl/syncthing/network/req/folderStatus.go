package req

import (
	"encoding/json"
	"fmt"
	"time"
)

func (p *SyncthingHttpClient) FolderStatus() (Model, error) {
	resp, err := p.get("rest/db/status?folder=" + p.folderName)
	if err != nil {
		return Model{}, err
	}

	var res Model
	if err := json.Unmarshal(resp, &res); err != nil {
		return Model{}, err
	}

	return res, nil
}

func (m *Model) StateChangedLog() string {
	return fmt.Sprintf(
		"State changed at: %v", m.StateChanged.Format("15:04:05"))
}

func (m *Model) hasError() bool {
	return m.State == "error" || m.State == "unknown"
}

func (m *Model) isIdle() bool {
	return m.State == "idle"
}

func (m *Model) OutOfSync() string {
	if m.NeedFiles > 0 {
		return fmt.Sprintf("There are %v remote files on workDir that are not exitst locally, you can click this icon to delete these remote extra files.", m.NeedFiles)
	} else {
		return ""
	}
}

type Model struct {
	GlobalBytes   int
	GlobalDeleted int
	GlobalFiles   int
	InSyncBytes   int
	InSyncFiles   int
	Invalid       string
	LocalBytes    int
	LocalDeleted  int
	LocalFiles    int
	NeedBytes     int
	NeedFiles     int
	State         string
	StateChanged  time.Time
	Version       int
}
