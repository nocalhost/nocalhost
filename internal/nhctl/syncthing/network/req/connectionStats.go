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

import "encoding/json"

func (p *SyncthingHttpClient) SystemConnections() (bool, error) {
	resp, err := p.get("rest/system/connections")
	if err != nil {
		return false, err
	}

	var res ConnectionStatsResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return false, err
	}

	if res.Connections == nil {
		return false, err
	}

	if val, ok := res.Connections[p.remoteDevice]; ok {
		return val.Connected, nil
	}
	return false, err
}

type ConnectionStatsResponse struct {
	Connections map[string]ConnectionStats `json:"connections"`
}

type ConnectionStats struct {
	Address       string
	Type          string
	Connected     bool
	Paused        bool
	ClientVersion string
	InBytesTotal  int64
	OutBytesTotal int64
}
