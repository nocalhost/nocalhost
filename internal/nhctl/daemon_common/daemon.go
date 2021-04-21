/*
Copyright 2021 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package daemon_common

import (
	"context"
)

const (
	DefaultDaemonPort = 30123
	SudoDaemonPort    = 30124
)

var (
	Version  = "1.0"
	CommitId = ""
)

type DaemonServerInfo struct {
	Version  string
	CommitId string
}

type PortForwardProfile struct {
	Cancel     context.CancelFunc `json:"-"` // For canceling a port forward
	StopCh     chan error         `json:"-"`
	NameSpace  string             `json:"nameSpace"`
	AppName    string             `json:"appName"`
	LocalPort  int                `json:"localPort"`
	RemotePort int                `json:"remotePort"`
}

func NewDaemonServerInfo() *DaemonServerInfo {
	return &DaemonServerInfo{Version: Version}
}

type CommonResponse struct {
	ErrInfo string `json:"errInfo"`
}

type DaemonServerStatusResponse struct {
	PortForwardList []*PortForwardProfile `json:"portForwardList"`
}
