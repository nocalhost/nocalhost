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

package profile

type AppProfileV2 struct {
	Name                    string            `json:"name" yaml:"name"`
	ChartName               string            `json:"chart_name" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	ReleaseName             string            `json:"release_name yaml:releaseName"`
	Namespace               string            `json:"namespace" yaml:"namespace"`
	Kubeconfig              string            `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string            `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 string            `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfileV2   `json:"svc_profile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool              `json:"installed" yaml:"installed"`
	SyncDirs                []string          `json:"syncDirs" yaml:"syncDirs"` // dev start -s
	ResourcePath            []string          `json:"resource_path" yaml:"resourcePath"`
	IgnoredPath             []string          `json:"ignoredPath" yaml:"ignoredPath"`
	PreInstall              []*PreInstallItem `json:"onPreInstall" yaml:"onPreInstall"`

	// After v2
	GitUrl       string `json:"gitUrl" yaml:"gitUrl"`
	GitRef       string `json:"gitRef" yaml:"gitRef"`
	HelmRepoUrl  string `json:"helmRepoUrl" yaml:"helmRepoUrl"`
	HelmRepoName string `json:"helmRepoUrl" yaml:"helmRepoName"`
	//HelmRepoChartVersion string `json:"helmRepoChartVersion" yaml:"helmRepoChartVersion"`

	Env     []*Env  `json:"env" yaml:"env"`
	EnvFrom EnvFrom `json:"envFrom" yaml:"envFrom"`
}

type SvcProfileV2 struct {
	*ServiceConfigV2 `yaml:"rawConfig"`
	ContainerProfile []*ContainerProfileV2 `json:"container_profile" yaml:"containerProfile"`
	ActualName       string                `json:"actual_name" yaml:"actualName"` // for helm, actualName may be ReleaseName-Name
	Developing       bool                  `json:"developing" yaml:"developing"`
	PortForwarded    bool                  `json:"port_forwarded" yaml:"portForwarded"`
	Syncing          bool                  `json:"syncing" yaml:"syncing"`
	SyncDirs         []string              `json:"syncDirs" yaml:"syncDirs,omitempty"` // dev start -s
	// same as local available port, use for port-forward
	RemoteSyncthingPort int `json:"remoteSyncthingPort" yaml:"remoteSyncthingPort"`
	// same as local available port, use for port-forward
	RemoteSyncthingGUIPort int    `json:"remoteSyncthingGUIPort" yaml:"remoteSyncthingGUIPort"`
	SyncthingSecret        string `json:"syncthingSecret" yaml:"syncthingSecret"` // secret name
	// syncthing local port
	LocalSyncthingPort                     int               `json:"localSyncthingPort" yaml:"localSyncthingPort"`
	LocalSyncthingGUIPort                  int               `json:"localSyncthingGUIPort" yaml:"localSyncthingGUIPort"`
	LocalAbsoluteSyncDirFromDevStartPlugin []string          `json:"localAbsoluteSyncDirFromDevStartPlugin" yaml:"localAbsoluteSyncDirFromDevStartPlugin"`
	DevPortForwardList                     []*DevPortForward `json:"devPortForwardList" yaml:"devPortForwardList"` // combine DevPortList,PortForwardStatusList and PortForwardPidList
	// Deprecated later
	DevPortList           []string `json:"devPortList" yaml:"devPortList"`
	PortForwardStatusList []string `json:"portForwardStatusList" yaml:"portForwardStatusList"`
	PortForwardPidList    []string `json:"portForwardPidList" yaml:"portForwardPidList"`
}

type ContainerProfileV2 struct {
	Name string
}

type DevPortForward struct {
	LocalPort  int
	RemotePort int
	Way        string
	Status     string
	Reason     string
	Updated    string
	Pid        int
}

// Compatible for v1
// Finding `containerName` config, if not found, use the first container config
func (s *SvcProfileV2) GetContainerDevConfigOrDefault(containerName string) *ContainerDevConfig {
	config := s.GetContainerDevConfig(containerName)
	if config == nil {
		config = s.GetDefaultContainerDevConfig()
	}
	return config
}

func (s *SvcProfileV2) GetDefaultContainerDevConfig() *ContainerDevConfig {
	//if s.ContainerConfigs[0].Name == "" {
	//	return s.ContainerConfigs[0].Dev
	//}
	if len(s.ContainerConfigs) == 0 {
		return nil
	}
	return s.ContainerConfigs[0].Dev
}

func (s *SvcProfileV2) GetContainerDevConfig(containerName string) *ContainerDevConfig {
	for _, devConfig := range s.ContainerConfigs {
		if devConfig.Name == containerName {
			return devConfig.Dev
		}
	}
	return nil
}
