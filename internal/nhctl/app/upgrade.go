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

import (
	"fmt"
	flag "nocalhost/internal/nhctl/app_flags"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
)

func (a *Application) Upgrade(installFlags flag.InstallFlags) error {

	switch a.GetType() {
	case HelmRepo:
		return a.upgradeForHelmRepo(installFlags)
	case Helm:
		return a.upgradeForHelmGit(installFlags)
	case Manifest:

	}

	return nil
}

func (a *Application) upgradeForManifest(installFlags flag.InstallFlags) error {

	err := a.downloadUpgradeResourcesFromGit(installFlags.GitUrl, installFlags.GitRef)
	if err != nil {
		return err
	}

	// Read upgrade resource obj
	getYamlFilesAndDirs()
	//a.client.GetResourceInfoFromFiles()

	return nil
}

func (a *Application) upgradeForHelmGit(installFlags flag.InstallFlags) error {

	err := a.downloadUpgradeResourcesFromGit(installFlags.GitUrl, installFlags.GitRef)
	if err != nil {
		return err
	}

	resourcesPath := a.getUpgradeResourceDir()
	releaseName := a.AppProfileV2.ReleaseName

	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "-n", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if installFlags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	params := []string{"upgrade", releaseName, resourcesPath[0]}
	if installFlags.HelmWait {
		params = append(params, "--wait")
	}

	params = append(params, "--timeout", "60m")
	params = append(params, commonParams...)

	//log.Info("building dependency...")
	//depParams := []string{"dependency", "build", resourcesPath[0]}
	//depParams = append(depParams, commonParams...)
	//_, err := tools.ExecCommand(nil, true, "helm", depParams...)
	//if err != nil {
	//	log.ErrorE(err, "fail to build dependency for helm app")
	//	return err
	//}

	fmt.Println("Upgrading helm application, this may take several minutes, please waiting...")
	_, err = tools.ExecCommand(nil, true, "helm", params...)
	if err != nil {
		return err
	}
	return a.saveUpgradeResources()
}

func (a *Application) upgradeForHelmRepo(installFlags flag.InstallFlags) error {

	releaseName := a.AppProfileV2.ReleaseName
	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "--namespace", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if installFlags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	chartName := installFlags.HelmChartName
	if a.configV2 != nil && a.configV2.ApplicationConfig.Name != "" {
		chartName = a.configV2.ApplicationConfig.Name
	}
	installParams := []string{"upgrade", releaseName}
	if installFlags.HelmWait {
		installParams = append(installParams, "--wait")
	}
	if installFlags.HelmRepoUrl != "" {
		installParams = append(installParams, chartName, "--repo", installFlags.HelmRepoUrl)
	} else if installFlags.HelmRepoName != "" {
		installParams = append(installParams, fmt.Sprintf("%s/%s", installFlags.HelmRepoName, chartName))
	}

	if installFlags.HelmRepoVersion != "" {
		installParams = append(installParams, "--version", installFlags.HelmRepoVersion)
	}

	installParams = append(installParams, "--timeout", "60m")
	installParams = append(installParams, commonParams...)

	log.Info("Upgrade helm application, this may take several minutes, please waiting...")

	_, err := tools.ExecCommand(nil, true, "helm", installParams...)
	return err
}
