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
		"Sync completed at: %v", m.StateChanged.Format("15:04:05"),
	)
}

func (m *Model) OutOfSyncLog() string {
	return fmt.Sprintf(
		"Out of sync! : %v ", m.StateChanged.Format("15:04:05"),
	)
}

func (m *Model) IdleTips() string {
	t := m.StateChanged.Format("15:04:05")

	return fmt.Sprintf(
		"The remote file is based on local sync dir at %v,"+
			" local changed before %v has been complete synchronized to the k8s sidecar.",
		t, t,
	)
}

func (m *Model) hasError() bool {
	return m.State == "error" || m.State == "unknown"
}

func (m *Model) isIdle() bool {
	return m.State == "idle"
}

func (m *Model) OutOfSyncTips() string {
	if m.NeedFiles > 0 {
		return fmt.Sprintf(
			"There are %v remote files on work dir that are different from locally "+
				"(may different or more than local), click the \"!\""+
				" on the left to hard reset remote according to local files.",
			m.NeedFiles,
		)
	} else {
		return ""
	}
}

func (m *Model) OutOfSync() string {
	if m.NeedFiles > 0 {
		return fmt.Sprintf(
			"There are %v remote files on work dir that are"+
				" different from locally (may different or more than local), "+
				"click me to hard reset remote according to local files.",
			m.NeedFiles,
		)
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
