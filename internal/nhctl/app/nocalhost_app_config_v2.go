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

package app

type NocalHostAppConfigV2 struct {
	ConfigProperties  *ConfigProperties  `json:"config_properties" yaml:"configProperties"`
	ApplicationConfig *ApplicationConfig `json:"application" yaml:"application"`
}

type ConfigProperties struct {
	Version string `json:"version" yaml:"version"`
	EnvFile string `json:"env_file" yaml:"envFile"`
}

type ApplicationConfig struct {
	Name           string             `json:"name" yaml:"name"`
	Type           AppType            `json:"manifestType" yaml:"manifestType"`
	ResourcePath   []string           `json:"resourcePath" yaml:"resourcePath"`
	IgnoredPath    []string           `json:"ignoredPath" yaml:"ignoredPath"`
	PreInstall     []*PreInstallItem  `json:"onPreInstall" yaml:"onPreInstall"`
	HelmValues     []*HelmValue       `json:"helm_values" yaml:"helmValues"`
	Env            []*Env             `json:"env" yaml:"env"`
	EnvFrom        *EnvFrom           `json:"env_from" yaml:"envFrom"`
	ServiceConfigs []*ServiceConfigV2 `json:"services" yaml:"services"`
}

type ServiceConfigV2 struct {
	Name                string               `json:"name" yaml:"name"`
	Type                SvcType              `json:"serviceType" yaml:"serviceType"`
	PriorityClass       string               `json:"priorityClass,omitempty" yaml:"priorityClass,omitempty"`
	DependLabelSelector *DependLabelSelector `json:"depend_label_selector" yaml:"dependLabelSelector"`
	ContainerConfigs    []*ContainerConfig   `json:"containers" yaml:"containers"`
}

type ContainerConfig struct {
	Name    string                  `json:"name" yaml:"name"`
	Install *ContainerInstallConfig `json:"install" yaml:"install"`
	Dev     *ContainerDevConfig     `json:"dev" yaml:"dev"`
}

type ContainerInstallConfig struct {
	Env         []*Env   `json:"env" yaml:"env"`
	EnvFrom     *EnvFrom `json:"env_from" yaml:"envFrom"`
	PortForward []string `json:"portForward" yaml:"portForward"`
}

type ContainerDevConfig struct {
	GitUrl                string                 `json:"git_url" yaml:"gitUrl"`
	Image                 string                 `json:"image" yaml:"image"`
	Shell                 string                 `json:"shell" yaml:"shell"`
	WorkDir               string                 `json:"work_dir" yaml:"workDir"`
	DevContainerResources *ResourceQuota         `json:"resources" yaml:"resources"`
	PersistentVolumeDirs  []*PersistentVolumeDir `json:"persistentVolumeDirs" yaml:"persistentVolumeDirs"`
	Command               *DevCommands           `json:"command" yaml:"command"`
	DebugConfig           *DebugConfig           `json:"debug" yaml:"debug"`
	UseDevContainer       bool                   `json:"useDevContainer" yaml:"useDevContainer"`
	Sync                  *SyncConfig            `json:"sync" yaml:"sync"`
	Env                   []*Env                 `json:"env" yaml:"env"`
	EnvFrom               *EnvFrom               `json:"env_from" yaml:"envFrom"`
	PortForward           []string               `json:"portForward" yaml:"portForward"`
}

type DevCommands struct {
	Build          []string `json:"build" yaml:"build"`
	Run            []string `json:"run" yaml:"run"`
	Debug          []string `json:"debug" yaml:"debug"`
	HotReloadRun   []string `json:"hotReloadRun" yaml:"hotReloadRun"`
	HotReloadDebug []string `json:"hotReloadDebug" yaml:"hotReloadDebug"`
}

type SyncConfig struct {
	Type              string   `json:"type" yaml:"type"`
	FilePattern       []string `json:"file_pattern" yaml:"filePattern"`
	IgnoreFilePattern []string `json:"ignore_file_pattern" yaml:"ignoreFilePattern"`
}

type DebugConfig struct {
	RemoteDebugPort string `json:"remote_debug_port" yaml:"remoteDebugPort"`
}

type DependLabelSelector struct {
	Pods []string `json:"pods" yaml:"pods"`
	Jobs []string `json:"jobs" yaml:"jobs"`
}

type HelmValue struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

type Env struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type EnvFrom struct {
	EnvFile []*EnvFile `json:"env_file" yaml:"envFile"`
}

type EnvFile struct {
	Path string `json:"path" yaml:"path"`
}
