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
	GetApplicationMeta    DaemonCommandType = "GetApplicationMeta"
	GetApplicationMetas   DaemonCommandType = "GetApplicationMetas"
	GetResourceInfo       DaemonCommandType = "GetResourceInfo"

	PREVIEW_VERSION = 0
	SUCCESS         = 200
	FAIL            = 400
	INTERNAL_FAIL   = 500
)

type BaseCommand struct {
	CommandType DaemonCommandType
	ClientStack string
	ClientPath  string
}

type BaseResponse struct {
	// zero for success
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   []byte `json:"data"`
}

type PortForwardCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	NameSpace   string `json:"nameSpace"`
	AppName     string `json:"appName"`
	Service     string `json:"service"`
	ServiceType string `json:"serviceType"`
	PodName     string `json:"podName"`
	LocalPort   int    `json:"localPort"`
	RemotePort  int    `json:"remotePort"`
	Role        string `json:"role"`
}

type GetApplicationMetaCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	NameSpace         string `json:"nameSpace"`
	AppName           string `json:"appName"`
	KubeConfigContent string `json:"kubeConfig"`
}

type GetApplicationMetasCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	NameSpace         string `json:"nameSpace"`
	KubeConfigContent string `json:"kubeConfig"`
}

type GetResourceInfoCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	KubeConfig   string            `json:"kubeConfig" yaml:"kubeConfig"`
	Namespace    string            `json:"namespace" yaml:"namespace"`
	AppName      string            `json:"appName" yaml:"appName"`
	Resource     string            `json:"resource" yaml:"resource"`
	ResourceName string            `json:"resourceName" yaml:"resourceName"`
	Label        map[string]string `json:"label" yaml:"label"`
}

func ParseBaseCommand(bys []byte) (DaemonCommandType, string, error) {
	base := &BaseCommand{}
	err := json.Unmarshal(bys, base)
	if err != nil {
		return "", "", errors.Wrap(err, "")
	}
	return base.CommandType, base.ClientStack, nil
}
