/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
