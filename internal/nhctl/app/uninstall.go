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
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
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
		//commonParams := make([]string, 0)
		//if a.GetNamespace() != "" {
		//	commonParams = append(commonParams, "--namespace", a.GetNamespace())
		//}
		//if a.AppProfile.Kubeconfig != "" {
		//	commonParams = append(commonParams, "--kubeconfig", a.AppProfile.Kubeconfig)
		//}
		//installParams := []string{"uninstall", a.Name}
		//installParams = append(installParams, commonParams...)
		//_, err := tools.ExecCommand(nil, true, "helm", installParams...)
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
