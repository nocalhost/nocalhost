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

// Deprecated
type NocalHostAppConfig struct {
	PreInstall   SortedRelPath `json:"onPreInstall" yaml:"onPreInstall"`
	ResourcePath RelPath       `json:"resourcePath" yaml:"resourcePath"`
	SvcConfigs   []*ServiceDevOptions  `json:"services" yaml:"services"`
	Name         string                `json:"name" yaml:"name"`
	Type         string                `json:"manifestType" yaml:"manifestType"`
	IgnoredPath  RelPath       `json:"ignoredPath" yaml:"ignoredPath"`
}

type PersistentVolumeDir struct {
	Path     string `json:"path" yaml:"path"`
	Capacity string `json:"capacity,omitempty" yaml:"capacity,omitempty"`
}

type ResourceQuota struct {
	Limits   *QuotaList `json:"limits" yaml:"limits"`
	Requests *QuotaList `json:"requests" yaml:"requests"`
}

type QuotaList struct {
	Memory string `json:"memory" yaml:"memory"`
	Cpu    string `json:"cpu" yaml:"cpu"`
}

type ServiceDevOptions struct {
	Name                  string                 `json:"name" yaml:"name"`
	Type                  string                 `json:"serviceType" yaml:"serviceType"`
	GitUrl                string                 `json:"gitUrl" yaml:"gitUrl"`
	DevImage              string                 `json:"devContainerImage" yaml:"devContainerImage"`
	WorkDir               string                 `json:"workDir" yaml:"workDir"`
	Sync                  []string               `json:"syncDirs" yaml:"syncDirs,omitempty"` // dev start -s
	PriorityClass         string                 `json:"priorityClass,omitempty" yaml:"priorityClass,omitempty"`
	PersistentVolumeDirs  []*PersistentVolumeDir `json:"persistentVolumeDirs" yaml:"persistentVolumeDirs"`
	BuildCommand          []string               `json:"buildCommand,omitempty" yaml:"buildCommand,omitempty"`
	RunCommand            []string               `json:"runCommand,omitempty" yaml:"runCommand,omitempty"`
	DebugCommand          []string               `json:"debugCommand,omitempty" yaml:"debugCommand,omitempty"`
	HotReloadRunCommand   []string               `json:"hotReloadRunCommand,omitempty" yaml:"hotReloadRunCommand,omitempty"`
	HotReloadDebugCommand []string               `json:"hotReloadDebugCommand,omitempty" yaml:"hotReloadDebugCommand,omitempty"`
	DevContainerShell     string                 `json:"devContainerShell" yaml:"devContainerShell"`
	DevContainerResources *ResourceQuota         `json:"devContainerResources" yaml:"devContainerResources"`
	DevPort               []string               `json:"devPorts" yaml:"devPorts"`
	Jobs                  []string               `json:"dependJobsLabelSelector" yaml:"dependJobsLabelSelector,omitempty"`
	Pods                  []string               `json:"dependPodsLabelSelector" yaml:"dependPodsLabelSelector,omitempty"`
	SyncedPattern         []string               `json:"syncFilePattern" yaml:"syncFilePattern"`
	IgnoredPattern        []string               `json:"ignoreFilePattern" yaml:"ignoreFilePattern"`
}
