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

package app_flags

type InstallFlags struct {
	GitUrl           string // resource url
	GitRef           string
	AppType          string
	HelmValueFile    string
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
}
