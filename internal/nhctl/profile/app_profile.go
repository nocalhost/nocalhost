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

// Deprecated: this struct is deprecated
type AppProfile struct {
	path                    string
	Name                    string        `json:"name" yaml:"name"`
	ChartName               string        `json:"chart_name" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	ReleaseName             string        `json:"release_name yaml:releaseName"`
	Namespace               string        `json:"namespace" yaml:"namespace"`
	Kubeconfig              string        `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string        `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 string        `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfile `json:"svc_profile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool          `json:"installed" yaml:"installed"`
	ResourcePath            []string      `json:"resource_path" yaml:"resourcePath"`
	IgnoredPath             []string      `json:"ignoredPath" yaml:"ignoredPath"`
}
