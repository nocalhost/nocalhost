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

package profile

type SvcProfile struct {
	*ServiceDevOptions `yaml:"rawConfig"`
	ActualName         string `json:"actual_name" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name
	Developing         bool   `json:"developing" yaml:"developing"`
	PortForwarded      bool   `json:"port_forwarded" yaml:"portForwarded"`
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
