/*
Copyright 2020 The Nocalhost Authors.
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

package profile

type SvcProfile struct {
	*ServiceDevOptions `         yaml:"rawConfig"`
	ActualName         string `yaml:"actualName"                             json:"actual_name"` // for helm, actualName may be ReleaseName-Name
	Developing         bool   `yaml:"developing"                             json:"developing"`
	PortForwarded      bool   `yaml:"portForwarded"                          json:"port_forwarded"`
	Syncing            bool   `yaml:"syncing"                                json:"syncing"`
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `yaml:"remoteSyncthingPort"                    json:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int    `yaml:"remoteSyncthingGUIPort"                 json:"remoteSyncthingGUIPort"`
	SyncthingSecret        string `yaml:"syncthingSecret"                        json:"syncthingSecret"` // secret name
	// syncthing local port
	LocalSyncthingPort                     int      `yaml:"localSyncthingPort"                     json:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int      `yaml:"localSyncthingGUIPort"                  json:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string `yaml:"localAbsoluteSyncDirFromDevStartPlugin" json:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortList                            []string `yaml:"devPortList"                            json:"devPortList"`
	PortForwardStatusList                  []string `yaml:"portForwardStatusList"                  json:"portForwardStatusList"`
	PortForwardPidList                     []string `yaml:"portForwardPidList"                     json:"portForwardPidList"`
	// .nhignore's pattern configuration
	SyncedPatterns  []string `yaml:"syncFilePattern"                        json:"syncFilePattern"`
	IgnoredPatterns []string `yaml:"ignoreFilePattern"                      json:"ignoreFilePattern"`
}
