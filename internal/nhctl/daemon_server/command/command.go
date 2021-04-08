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

package command

import (
	"encoding/json"
	"github.com/pkg/errors"
)

type DaemonCommandType string

const (
	StartPortForward      DaemonCommandType = "StartPortForward"
	RestartPortForward    DaemonCommandType = "RestartPortForward"
	StopPortForward       DaemonCommandType = "StopPortForward"
	StopDaemonServer      DaemonCommandType = "StopDaemonServer"
	RestartDaemonServer   DaemonCommandType = "RestartDaemonServer"
	GetDaemonServerInfo   DaemonCommandType = "GetDaemonServerInfo"
	GetDaemonServerStatus DaemonCommandType = "GetDaemonServerStatus"
)

type BaseCommand struct {
	CommandType DaemonCommandType
}

//type DaemonServerCommand struct {
//	CommandType DaemonCommandType
//	IsSudo      bool
//}

type PortForwardCommand struct {
	CommandType DaemonCommandType
	NameSpace   string `json:"nameSpace"`
	AppName     string `json:"appName"`
	Service     string `json:"service"`
	PodName     string `json:"podName"`
	LocalPort   int    `json:"localPort"`
	RemotePort  int    `json:"remotePort"`
	Role        string `json:"role"`
	//IsSudo      bool   `json:"isSudo"` // todo: move to profile?
}

//type EndPortForwardCommand struct {
//	NameSpace  string `json:"nameSpace"`
//	AppName    string `json:"appName"`
//	LocalPort  int    `json:"localPort"`
//	RemotePort int    `json:"remotePort"`
//}

func ParseCommandType(bys []byte) (DaemonCommandType, error) {
	base := &BaseCommand{}
	err := json.Unmarshal(bys, base)
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	return base.CommandType, nil
}
