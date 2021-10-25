/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package req

import (
	"encoding/json"
	"fmt"
)

func (p *SyncthingHttpClient) Completion() (FolderCompletion, error) {

	resp, err := p.get(fmt.Sprintf("rest/db/completion?device=%s&folder=%s", p.remoteDevice, p.folderName))
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
	var compl string
	if f == nil || 1 > f.Completion {
		compl = "0.00%"
	} else {
		compl = fmt.Sprintf("%.2f%%", f.Completion)
	}

	return fmt.Sprintf(
		"Upload to remote: %s", compl,
	)
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
