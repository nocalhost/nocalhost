/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
