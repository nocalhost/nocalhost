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
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/resource"
	flag "nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"os"
)

func (a *Application) Upgrade(installFlags *flag.InstallFlags) error {

	switch a.GetType() {
	case HelmRepo:
		return a.upgradeForHelmRepo(installFlags)
	case Helm:
		return a.upgradeForHelmGitOrHelmLocal(installFlags)
	case Manifest:
		return a.upgradeForManifest(installFlags)
	case ManifestLocal:
		return a.upgradeForManifest(installFlags)
	case HelmLocal:
		return a.upgradeForHelmGitOrHelmLocal(installFlags)
	case KustomizeGit:
		return a.upgradeForKustomize()
	default:
		return errors.New("Unsupported app type")
	}

}

func (a *Application) upgradeForKustomize() error {
	resourcesPath := a.GetResourceDir()
	if len(resourcesPath) > 1 {
		log.Warn(`There are multiple resourcesPath settings, will use first one`)
	}
	useResourcePath := resourcesPath[0]
	err := a.client.ApplyForCreate([]string{}, true, StandardNocalhostMetas(a.Name, a.GetNamespace()), useResourcePath)
	if err != nil {
		return err
	}
	return nil
}

func (a *Application) upgradeForManifest(installFlags *flag.InstallFlags) error {

	var err error
	if installFlags.GitUrl != "" {
		err = a.downloadUpgradeResourcesFromGit(installFlags.GitUrl, installFlags.GitRef)
		if err != nil {
			return err
		}
	} else if installFlags.LocalPath != "" {
		err = a.copyUpgradeResourcesFromLocalDir(installFlags.LocalPath)
		if err != nil {
			return errors.Wrap(err, "")
		}
	} else {
		return errors.New("LocalPath or GitUrl mush be specified")
	}

	var upgradeResourcePath []string
	if len(installFlags.ResourcePath) > 0 {
		upgradeResourcePath = installFlags.ResourcePath
	} else {
		// Get resource path for upgrade .nocalhost
		configFilePath := a.getUpgradeConfigPathInGitResourcesDir(installFlags.Config)
		_, err := os.Stat(configFilePath)
		if err != nil {
			log.WarnE(errors.Wrap(err, ""), "Failed to load config.yaml")
		} else {
			// Render
			configFile := fp.NewFilePath(configFilePath)

			var envFile *fp.FilePathEnhance
			if relPath := gettingRenderEnvFile(configFilePath); relPath != "" {
				envFile = configFile.RelOrAbs("../").RelOrAbs(relPath)

				if e := envFile.CheckExist(); e != nil {
					log.Log("Render %s Nocalhost config without env files, we found the env file had been configured as %s, but we can not found in %s", configFile.Abs(), relPath, envFile.Abs())
				} else {
					log.Log("Render %s Nocalhost config with env files %s", configFile.Abs(), envFile.Abs())
				}
			} else {
				log.Log("Render %s Nocalhost config without env files, you config your Nocalhost configuration such as: \nconfigProperties:\n  envFile: ./envs/env\n  version: v2", configFile.Abs())
			}

			renderedStr, err := envsubst.Render(configFile, envFile)
			//configBytes, err := ioutil.ReadFile(configFilePath)
			if err != nil {
				log.WarnE(errors.Wrap(err, ""), err.Error())
			} else {
				configBytes := []byte(renderedStr)
				// render config bytes
				configV2 := &profile.NocalHostAppConfigV2{}
				err = yaml.Unmarshal(configBytes, configV2)
				if err != nil {
					log.WarnE(errors.Wrap(err, ""), "Failed to unmarshal config v2")
				} else {
					if configV2.ConfigProperties != nil && configV2.ConfigProperties.Version == "v2" {
						if configV2.ApplicationConfig != nil && len(configV2.ApplicationConfig.ResourcePath) > 0 {
							log.Info("Updating resource path from config v2")
							upgradeResourcePath = configV2.ApplicationConfig.ResourcePath
						}
					} else {
						configV1 := &profile.NocalHostAppConfig{}
						err = yaml.Unmarshal(configBytes, configV1)
						if err != nil {
							log.WarnE(errors.Wrap(err, ""), "Failed to unmarshal config v1")
						} else {
							if len(configV1.ResourcePath) > 0 {
								log.Info("Updating resource path from config v1")
								upgradeResourcePath = configV1.ResourcePath
							}
						}

					}
				}
			}
		}
	}

	// Read upgrade resource obj
	a.loadUpgradePreInstallAndInstallManifest(upgradeResourcePath)
	upgradeInfos, err := a.client.GetResourceInfoFromFiles(a.upgradeInstallManifest, true, "")
	if err != nil {
		return err
	}

	// Read current resource obj
	a.loadPreInstallAndInstallManifest()
	oldInfos, err := a.client.GetResourceInfoFromFiles(a.installManifest, true, "")
	if err != nil {
		return err
	}

	err = a.upgradeInfos(oldInfos, upgradeInfos, true)
	if err != nil {
		return err
	}
	if len(upgradeResourcePath) > 0 {
		appProfile, _ := a.GetProfile()
		appProfile.ResourcePath = upgradeResourcePath
		_ = a.SaveProfile(appProfile)
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
		err := a.client.ApplyResourceInfo(info, StandardNocalhostMetas(a.Name, a.GetNamespace()))
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
		err := a.client.ApplyResourceInfo(info, StandardNocalhostMetas(a.Name, a.GetNamespace()))
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

func (a *Application) upgradeForHelmGitOrHelmLocal(installFlags *flag.InstallFlags) error {

	var err error
	if installFlags.GitUrl != "" {
		err = a.downloadUpgradeResourcesFromGit(installFlags.GitUrl, installFlags.GitRef)
		if err != nil {
			return err
		}
	} else if installFlags.LocalPath != "" {
		err = a.copyUpgradeResourcesFromLocalDir(installFlags.LocalPath)
		if err != nil {
			return errors.Wrap(err, "")
		}
	} else {
		return errors.New("LocalPath or GitUrl mush be specified")
	}

	appProfile, err := a.GetProfile()
	if err != nil {
		return err
	}
	resourcesPath := a.getUpgradeResourceDir(appProfile.ResourcePath)
	releaseName := appProfile.ReleaseName

	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "-n", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	//if installFlags.Debug {
	//commonParams = append(commonParams, "--debug")
	//}

	params := []string{"upgrade", releaseName, resourcesPath[0]}
	if installFlags.HelmWait {
		params = append(params, "--wait")
	}

	params = append(params, "--timeout", "60m")
	params = append(params, commonParams...)

	log.Info("building dependency...")
	depParams := []string{"dependency", "build", resourcesPath[0]}
	depParams = append(depParams, commonParams...)
	_, err = tools.ExecCommand(nil, true, "helm", depParams...)
	if err != nil {
		log.ErrorE(err, "fail to build dependency for helm app")
		return err
	}

	fmt.Println("Upgrading helm application, this may take several minutes, please waiting...")
	_, err = tools.ExecCommand(nil, true, "helm", params...)
	if err != nil {
		return err
	}
	return a.saveUpgradeResources()
}

func (a *Application) upgradeForHelmRepo(installFlags *flag.InstallFlags) error {

	appProfile, err := a.GetProfile()
	if err != nil {
		return err
	}
	releaseName := appProfile.ReleaseName
	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "--namespace", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	//if installFlags.Debug {
	//	commonParams = append(commonParams, "--debug")
	//}

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

	_, err = tools.ExecCommand(nil, true, "helm", installParams...)
	return err
}
