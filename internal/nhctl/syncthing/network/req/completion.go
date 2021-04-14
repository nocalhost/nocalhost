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
