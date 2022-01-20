/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package app_flags

type InstallFlags struct {
	GitUrl           string // resource url
	GitRef           string
	AppType          string
	HelmValueFile    []string
	ForceInstall     bool
	IgnorePreInstall bool
	HelmSet          []string
	HelmRepoName     string
	HelmRepoUrl      string
	HelmRepoVersion  string
	HelmChartName    string
	HelmWait         bool
	OuterConfig      string
	Config           string
	ResourcePath     []string
	//Namespace        string
	LocalPath string
}

type ListFlags struct {
	Yaml bool
	Json bool
	Full bool
}
