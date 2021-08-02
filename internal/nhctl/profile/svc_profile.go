/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package profile

type SvcProfile struct {
	*ServiceDevOptions `yaml:"rawConfig"`
	ActualName         string `json:"actualName" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name
	Developing         bool   `json:"developing" yaml:"developing"`
	PortForwarded      bool   `json:"portForwarded" yaml:"portForwarded"`
	Syncing            bool   `json:"syncing" yaml:"syncing"`
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int    `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	SyncthingSecret        string `json:"syncthingSecret" yaml:"syncthingSecret"` // secret name
	// syncthing local port
	LocalSyncthingPort                     int      `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int      `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortList                            []string `json:"devPortList" yaml:"devPortList"`
	PortForwardStatusList                  []string `json:"portForwardStatusList" yaml:"portForwardStatusList"`
	PortForwardPidList                     []string `json:"portForwardPidList" yaml:"portForwardPidList"`
	// .nhignore's pattern configuration
	SyncedPatterns  []string `json:"syncFilePattern" yaml:"syncFilePattern"`
	IgnoredPatterns []string `json:"ignoreFilePattern" yaml:"ignoreFilePattern"`
}
