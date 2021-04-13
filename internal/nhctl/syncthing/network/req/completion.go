package req

import (
	"encoding/json"
	"fmt"
)

func (p *SyncthingHttpClient) Completion() (FolderCompletion, error) {

	resp, err := p.get(
		fmt.Sprintf("rest/db/completion?device=%s&folder=%s", p.remoteDevice, p.folderName),
	)
	if err != nil {
		return FolderCompletion{}, err
	}

	var comp FolderCompletion
	if err := json.Unmarshal(resp, &comp); err != nil {
		return comp, err
	}

	return comp, nil
}

func (f *FolderCompletion) NeedOverrideForce() bool {
	return f.Completion == 95
}

func (f *FolderCompletion) isComplete() bool {
	return f.Completion == 100
}

func (f *FolderCompletion) UploadPct() string {
	return fmt.Sprintf(
		"Upload to remote: %.2f%%", f.Completion)
}

type FolderCompletion struct {
	Completion  float64 `json:"completion"`
	GlobalBytes int64   `json:"globalBytes"`
	NeedBytes   int64   `json:"needBytes"`
	GlobalItems int     `json:"globalItems"`
	NeedItems   int     `json:"needItems"`
	NeedDeletes int     `json:"needDeletes"`
	Sequence    int64   `json:"sequence"`
}
