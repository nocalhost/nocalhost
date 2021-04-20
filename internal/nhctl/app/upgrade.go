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

package app

import (
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/resource"
	flag "nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"os"
)

func (a *Application) PrepareForUpgrade(installFlags *flag.InstallFlags) error {

	var err error
	if installFlags.GitUrl != "" {
		if err = a.downloadUpgradeResourcesFromGit(installFlags.GitUrl, installFlags.GitRef); err != nil {
			return err
		}
	} else if installFlags.LocalPath != "" {
		if err = a.copyUpgradeResourcesFromLocalDir(installFlags.LocalPath); err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (a *Application) Upgrade(installFlags *flag.InstallFlags) error {

	switch a.GetType() {
	case appmeta.HelmRepo:
		return a.upgradeForHelm(installFlags, true)
	case appmeta.Helm, appmeta.HelmLocal:
		return a.upgradeForHelm(installFlags, false)
	case appmeta.Manifest, appmeta.ManifestLocal:
		return a.upgradeForManifest(installFlags)
	case appmeta.KustomizeGit:
		return a.upgradeForKustomize(installFlags)
	default:
		return errors.New("Unsupported app type")
	}
}

func (a *Application) upgradeForKustomize(installFlags *flag.InstallFlags) error {
	var err error
	resourcesPath := a.getUpgradeResourceDir(installFlags.ResourcePath)
	if len(resourcesPath) > 1 {
		log.Warn(`There are multiple resourcesPath settings, will use first one`)
	}
	useResourcePath := resourcesPath[0]

	err = a.client.Apply(
		[]string{}, true,
		StandardNocalhostMetas(a.Name, a.NameSpace).SetBeforeApply(
			func(manifest string) error {
				a.appMeta.Manifest = manifest
				return a.appMeta.Update()
			},
		),
		useResourcePath,
	)
	if err != nil {
		return err
	}
	return moveDir(a.getUpgradeGitDir(), a.ResourceTmpDir)
}

func (a *Application) upgradeForManifest(installFlags *flag.InstallFlags) error {

	var err error
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
					log.Log(
						"Render %s Nocalhost config without env files, we found the env "+
							"file had been configured as %s, but we can not found in %s",
						configFile.Abs(), relPath, envFile.Abs(),
					)
				} else {
					log.Log("Render %s Nocalhost config with env files %s", configFile.Abs(), envFile.Abs())
				}
			} else {
				log.Log(
					"Render %s Nocalhost config without env files, you config your Nocalhost "+
						"configuration such as: \nconfigProperties:\n  envFile: ./envs/env\n  version: v2",
					configFile.Abs(),
				)
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

	profileV2, err := a.GetProfile()
	if err != nil {
		return err
	}
	// todo need to refactor
	profileV2.ResourcePath = upgradeResourcePath
	_, manifests := profileV2.LoadManifests(a.getUpgradeGitDir())

	// Read upgrade resource obj
	updateResource, err := clientgoutils.NewManifestResourceReader(manifests).LoadResource()
	if err != nil {
		return err
	}

	upgradeInfos, err := updateResource.GetResourceInfo(a.client, true)
	if err != nil {
		return err
	}

	// Read current resource obj
	oldInfos, err := a.appMeta.NewResourceReader().GetResourceInfo(a.client, true)
	if err != nil {
		return err
	}

	if err = a.upgradeInfos(oldInfos, upgradeInfos, true); err != nil {
		return err
	}

	a.appMeta.Manifest = updateResource.String()
	if err := a.appMeta.Update(); err != nil {
		return err
	}

	return moveDir(a.getUpgradeGitDir(), a.ResourceTmpDir)
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
			log.WarnE(err, fmt.Sprintf("Failed to delete resource %s , Err: %s ", info.Name, err.Error()))
			if !continueOnErr {
				return err
			}
		}
	}

	for _, info := range infosToCreate {
		log.Infof("Creating resource(%s) %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
		err := a.client.ApplyResourceInfo(info, StandardNocalhostMetas(a.Name, a.NameSpace))
		if err != nil {
			log.WarnE(err, fmt.Sprintf("Failed to create resource %s", info.Name))
			if !continueOnErr {
				return err
			}
		}
	}

	for _, info := range infosToUpdate {
		log.Infof("Updating resource(%s) %s", info.Object.GetObjectKind().GroupVersionKind().Kind, info.Name)
		err := a.client.ApplyResourceInfo(info, StandardNocalhostMetas(a.Name, a.NameSpace))
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
		if in.Name == info.Name && in.Object.GetObjectKind().GroupVersionKind() ==
			info.Object.GetObjectKind().GroupVersionKind() {
			return true
		}
	}
	return false
}

func (a *Application) upgradeForHelm(installFlags *flag.InstallFlags, fromRepo bool) error {

	appProfile, err := a.GetProfile()
	if err != nil {
		return err
	}
	releaseName := appProfile.ReleaseName

	commonParams := make([]string, 0)
	if a.NameSpace != "" {
		commonParams = append(commonParams, "--namespace", a.NameSpace)
	}
	if a.KubeConfig != "" {
		commonParams = append(commonParams, "--kubeconfig", a.KubeConfig)
	}

	params := []string{"upgrade", releaseName}

	if fromRepo {
		chartName := installFlags.HelmChartName
		if a.appMeta.Config != nil && a.appMeta.Config.ApplicationConfig.Name != "" {
			chartName = a.appMeta.Config.ApplicationConfig.Name
		}
		if installFlags.HelmRepoUrl != "" {
			params = append(params, chartName, "--repo", installFlags.HelmRepoUrl)
		} else if installFlags.HelmRepoName != "" {
			params = append(params, fmt.Sprintf("%s/%s", installFlags.HelmRepoName, chartName))
		}
		if installFlags.HelmRepoVersion != "" {
			params = append(params, "--version", installFlags.HelmRepoVersion)
		}
	} else {
		resourcesPath := a.getUpgradeResourceDir(appProfile.ResourcePath)
		params = append(params, resourcesPath[0])
		log.Info("building dependency...")
		depParams := []string{"dependency", "build", resourcesPath[0]}
		depParams = append(depParams, commonParams...)
		if _, err = tools.ExecCommand(nil, true, false, "helm", depParams...);
			err != nil {
			return errors.Wrap(err, "fail to build dependency for helm app")
		}
	}

	if installFlags.HelmWait {
		params = append(params, "--wait")
	}
	params = append(params, "--timeout", "60m")
	params = append(params, commonParams...)

	log.Info("Upgrade helm application, this may take several minutes, please waiting...")

	if _, err = tools.ExecCommand(nil, true, false, "helm", params...);
		err != nil {
		return errors.Wrap(err, "")
	}

	if !fromRepo {
		err = a.saveUpgradeResources()
	}
	return err
}
