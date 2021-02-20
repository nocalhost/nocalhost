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
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/resource"
	flag "nocalhost/internal/nhctl/app_flags"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
)

func (a *Application) Upgrade(installFlags *flag.InstallFlags) error {

	switch a.GetType() {
	case HelmRepo:
		return a.upgradeForHelmRepo(installFlags)
	case Helm:
		return a.upgradeForHelmGit(installFlags)
	case Manifest:
		return a.upgradeForManifest(installFlags)
	default:
		return errors.New("Unsupported app type")
	}

}

func (a *Application) upgradeForManifest(installFlags *flag.InstallFlags) error {

	err := a.downloadUpgradeResourcesFromGit(installFlags.GitUrl, installFlags.GitRef)
	if err != nil {
		return err
	}

	// Read upgrade resource obj
	a.loadUpgradePreInstallAndInstallManifest()
	upgradeInfos, err := a.client.GetResourceInfoFromFiles(a.upgradeInstallManifest, true)
	if err != nil {
		return err
	}

	// Read current resource obj
	a.loadPreInstallAndInstallManifest()
	oldInfos, err := a.client.GetResourceInfoFromFiles(a.installManifest, true)
	if err != nil {
		return err
	}

	err = a.upgradeInfos(oldInfos, upgradeInfos, true)
	if err != nil {
		return err
	}

	return moveDir(a.getUpgradeGitDir(), a.getGitDir())
}

func (a *Application) upgradeInfos(oldInfos []*resource.Info, upgradeInfos []*resource.Info, continueOnErr bool) error {

	infosToDelete := make([]*resource.Info, 0)
	infosToCreate := make([]*resource.Info, 0)
	infosToUpdate := make([]*resource.Info, 0)

	// If a resource defined in oldInfos, but not in upgradeInfos, delete it
	for _, info := range oldInfos {
		if !isContainsInfo(info, upgradeInfos) {
			infosToDelete = append(infosToDelete, info)
		}
	}

	// If a resource defined in upgradeInfos, but not in oldInfos, create it
	// If a resource defined both in upgradeInfos and oldInfos, update it
	for _, info := range upgradeInfos {
		if !isContainsInfo(info, oldInfos) {
			infosToCreate = append(infosToCreate, info)
		} else {
			infosToUpdate = append(infosToUpdate, info)
		}
	}

	for _, info := range infosToDelete {
		log.Infof("Deleting resource(%s) %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
		err := a.client.DeleteResourceInfo(info)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("Failed to delete resource %s", info.Name))
			if !continueOnErr {
				return err
			}
		}
	}

	for _, info := range infosToCreate {
		log.Infof("Creating resource(%s) %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
		err := a.client.CreateResourceInfo(info)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("Failed to create resource %s", info.Name))
			if !continueOnErr {
				return err
			}
		}
	}

	for _, info := range infosToUpdate {
		log.Infof("Updating resource(%s) %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
		//err := a.client.UpdateResourceInfoByServerSide(info)
		err := a.client.ApplyResourceInfo(info)
		if err != nil {
			log.WarnE(err, fmt.Sprintf("Failed to create resource %s", info.Name))
			if !continueOnErr {
				return err
			}
		}
	}
	return nil
}

func isContainsInfo(info *resource.Info, infos []*resource.Info) bool {
	if info == nil || len(infos) == 0 {
		return false
	}
	for _, in := range infos {
		if in.Name == info.Name && in.Object.GetObjectKind().GroupVersionKind() == info.Object.GetObjectKind().GroupVersionKind() {
			return true
		}
	}
	return false
}

func (a *Application) upgradeForHelmGit(installFlags *flag.InstallFlags) error {

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

func (a *Application) upgradeForHelmRepo(installFlags *flag.InstallFlags) error {

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
