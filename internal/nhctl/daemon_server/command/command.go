/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package command

import (
	"encoding/json"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
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
	UpdateApplicationMeta DaemonCommandType = "UpdateApplicationMeta"
	KubeconfigOperation   DaemonCommandType = "KubeconfigOperationCommand"
	CheckClusterStatus    DaemonCommandType = "CheckClusterStatus"
	FlushDirMappingCache  DaemonCommandType = "FlushDirMappingCache"
	AuthCheck             DaemonCommandType = "AuthCheck"

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
	Nid         string `json:"nid"`
}

type GetApplicationMetaCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	NameSpace         string `json:"nameSpace"`
	AppName           string `json:"appName"`
	KubeConfigContent string `json:"kubeConfig"`
}

type CheckClusterStatusCommand struct {
	CommandType DaemonCommandType
	ClientStack string

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
	ShowHidden   bool              `json:"showHidden" yaml:"showHidden"`
}

type AuthCheckCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	KubeConfigContent string   `json:"kubeConfig" yaml:"kubeConfig"`
	NameSpace         string   `json:"namespace" yaml:"namespace"`
	NeedChecks        []string `json:"needChecks" yaml:"needChecks"`
}

type UpdateApplicationMetaCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	KubeConfig string     `json:"kubeConfig" yaml:"kubeConfig"`
	Namespace  string     `json:"namespace" yaml:"namespace"`
	SecretName string     `json:"secretName" yaml:"secretName"`
	Secret     *v1.Secret `json:"secret" yaml:"secret"`
}

type KubeconfigOperationCommand struct {
	CommandType DaemonCommandType
	ClientStack string

	KubeConfigBytes []byte    `json:"kubeConfigBytes" yaml:"kubeConfigBytes"`
	Namespace       string    `json:"namespace" yaml:"namespace"`
	Operation       Operation `json:"operation" yaml:"operation"`
}

type Operation string

const (
	OperationAdd    Operation = "add"
	OperationRemove Operation = "remove"
)

func ParseBaseCommand(bys []byte) (DaemonCommandType, string, error) {
	base := &BaseCommand{}
	err := json.Unmarshal(bys, base)
	if err != nil {
		return "", "", errors.Wrap(err, "")
	}
	return base.CommandType, base.ClientStack, nil
}
