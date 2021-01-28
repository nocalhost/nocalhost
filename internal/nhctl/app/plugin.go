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

package app

type PluginGetApplication struct {
	Name                    string                 `json:"name" yaml:"name"`
	ReleaseName             string                 `json:"release_name yaml:releaseName"`
	Namespace               string                 `json:"namespace" yaml:"namespace"`
	Kubeconfig              string                 `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string                 `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType                `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfileForPlugin `json:"svc_profile" yaml:"svcProfile"` // this will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool                   `json:"installed" yaml:"installed"`
	ResourcePath            []string               `json:"resource_path" yaml:"resourcePath"`
}

type PluginGetApplicationService struct {
	Name                    string               `json:"name" yaml:"name"`
	ReleaseName             string               `json:"release_name yaml:releaseName"`
	Namespace               string               `json:"namespace" yaml:"namespace"`
	Kubeconfig              string               `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string               `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType              `json:"app_type" yaml:"appType"`
	SvcProfile              *SvcProfileForPlugin `json:"svc_profile" yaml:"svcProfile"` // this will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool                 `json:"installed" yaml:"installed"`
	ResourcePath            []string             `json:"resource_path" yaml:"resourcePath"`
}
