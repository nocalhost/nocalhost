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
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"strings"
)

func (a *Application) cleanUpDepConfigMap() error {

	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	if profileV2.DependencyConfigMapName != "" {
		log.Debugf("Cleaning up config map %s", profileV2.DependencyConfigMapName)
		err := a.client.DeleteConfigMapByName(profileV2.DependencyConfigMapName)
		if err != nil {
			return err
		}
		profileV2.DependencyConfigMapName = ""
		profileV2.Save()
	} else {
		log.Debug("No dependency config map needs to clean up")
	}

	// Clean up all dep config map
	list, err := a.client.GetConfigMaps()
	if err != nil {
		return err
	}

	for _, c := range list {
		if strings.HasPrefix(c.Name, DependenceConfigMapPrefix) {
			err = a.client.DeleteConfigMapByName(c.Name)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("Failed to clean up config map: %s", c.Name))
			}
		}
	}
	return nil
}

func (a *Application) uninstallHelm() error {

	if !a.IsHelm() {
		log.Debugf("This is not a helm application")
		return nil
	}

	commonParams := make([]string, 0)
	if a.NameSpace != "" {
		commonParams = append(commonParams, "--namespace", a.NameSpace)
	}
	appProfile, _ := a.GetProfile()
	if appProfile.Kubeconfig != "" {
		commonParams = append(commonParams, "--kubeconfig", appProfile.Kubeconfig)
	}
	uninstallParams := []string{"uninstall", a.Name}
	uninstallParams = append(uninstallParams, commonParams...)
	_, err := tools.ExecCommand(nil, true, "helm", uninstallParams...)
	return err
}

func (a *Application) UninstallKustomize() error {
	resourcesPath := a.GetResourceDir()
	if len(resourcesPath) > 1 {
		log.Warn(`There are multiple resourcesPath settings, will use first one`)
	}
	useResourcePath := resourcesPath[0]
	commonParams := make([]string, 0)
	if a.NameSpace != "" {
		commonParams = append(commonParams, "--namespace", a.NameSpace)
	}
	appProfile, _ := a.GetProfile()
	if appProfile.Kubeconfig != "" {
		commonParams = append(commonParams, "--kubeconfig", appProfile.Kubeconfig)
	}
	uninstallParams := []string{"delete", "-k", useResourcePath}
	uninstallParams = append(uninstallParams, commonParams...)
	_, err := tools.ExecCommand(nil, true, "kubectl", uninstallParams...)
	return err
}

func (a *Application) Uninstall(force bool) error {

	err := a.cleanUpDepConfigMap()
	if err != nil && !force {
		return err
	}

	if a.IsHelm() {
		err = a.uninstallHelm()
		if err != nil && !force {
			return err
		}
	} else if a.IsManifest() {
		a.loadPreInstallAndInstallManifest()
		a.cleanPreInstall()
		err := a.uninstallManifestRecursively()
		if err != nil {
			return err
		}
	} else if a.IsKustomize() {
		err = a.UninstallKustomize()
		if err != nil {
			return err
		}
	}

	err = a.CleanupResources()
	if err != nil && !force {
		return err
	}

	return nil
}
