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

import (
	"fmt"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"strings"
)

func (a *Application) cleanUpDepConfigMap() error {

	if a.AppProfile.DependencyConfigMapName != "" {
		log.Debugf("Cleaning up config map %s", a.AppProfile.DependencyConfigMapName)
		err := a.client.DeleteConfigMapByName(a.AppProfile.DependencyConfigMapName)
		if err != nil {
			return err
		}
		a.AppProfile.DependencyConfigMapName = ""
		a.AppProfile.Save()
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
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "--namespace", a.GetNamespace())
	}
	if a.AppProfile.Kubeconfig != "" {
		commonParams = append(commonParams, "--kubeconfig", a.AppProfile.Kubeconfig)
	}
	uninstallParams := []string{"uninstall", a.Name}
	uninstallParams = append(uninstallParams, commonParams...)
	_, err := tools.ExecCommand(nil, true, "helm", uninstallParams...)
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
		a.cleanPreInstall()
		err := a.uninstallManifestRecursively()
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
