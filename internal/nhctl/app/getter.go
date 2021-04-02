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

import "nocalhost/pkg/nhctl/clientgoutils"

func (a *Application) GetType() AppType {
	appProfile, _ := a.GetProfile()
	return AppType(appProfile.AppType)
}

func (a *Application) GetKubeconfig() string {
	appProfile, _ := a.GetProfile()
	return appProfile.Kubeconfig
}

func (a *Application) IsHelm() bool {
	appProfile, _ := a.GetProfile()
	return appProfile.AppType == string(Helm) || appProfile.AppType == string(HelmRepo) || appProfile.AppType == string(HelmLocal)
}

func (a *Application) IsManifest() bool {
	appProfile, _ := a.GetProfile()
	return appProfile.AppType == string(Manifest) || appProfile.AppType == string(ManifestLocal)
}

func (a *Application) IsKustomize() bool {
	appProfile, _ := a.GetProfile()
	return appProfile.AppType == string(KustomizeGit)
}

func (a *Application) GetClient() *clientgoutils.ClientGoUtils {
	return a.client
}
