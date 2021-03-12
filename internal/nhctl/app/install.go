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
	"context"
	"fmt"
	"math/rand"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
)

func (a *Application) Install(ctx context.Context, flags *HelmFlags) error {

	err := a.InstallDepConfigMap()
	if err != nil {
		return errors.Wrap(err, "failed to install dep config map")
	}

	switch a.AppProfileV2.AppType {
	case Helm:
		err = a.installHelmInGit(flags)
	case HelmRepo:
		err = a.installHelmInRepo(flags)
	case Manifest:
		err = a.InstallManifest()
	case ManifestLocal:
		err = a.InstallManifest()
	case HelmLocal:
		err = a.installHelmInGit(flags)
	default:
		return errors.New(fmt.Sprintf("unsupported application type, must be %s, %s or %s", Helm, HelmRepo, Manifest))
	}
	if err != nil {
		a.cleanUpDepConfigMap() // clean up dep config map

		// Clean up helm release after failed
		if a.IsHelm() {
			a.uninstallHelm()
		}
		return err
	}

	a.SetInstalledStatus(true)
	return nil
}

func (a *Application) InstallManifest() error {
	a.loadPreInstallAndInstallManifest()
	a.preInstall()

	// install manifest recursively, don't install pre-install workload again
	err := a.installManifestRecursively()
	return errors.Wrap(err, "")
}

func (a *Application) installHelmInRepo(flags *HelmFlags) error {

	releaseName := a.Name
	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "--namespace", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	chartName := flags.Chart
	if a.configV2 != nil && a.configV2.ApplicationConfig.Name != "" {
		chartName = a.configV2.ApplicationConfig.Name
	}
	installParams := []string{"install", releaseName}
	if flags.Wait {
		installParams = append(installParams, "--wait")
	}
	if flags.RepoUrl != "" {
		installParams = append(installParams, chartName, "--repo", flags.RepoUrl)
		a.AppProfileV2.HelmRepoUrl = flags.RepoUrl
	} else if flags.RepoName != "" {
		installParams = append(installParams, fmt.Sprintf("%s/%s", flags.RepoName, chartName))
		a.AppProfileV2.HelmRepoName = flags.RepoName
	}
	if flags.Version != "" {
		installParams = append(installParams, "--version", flags.Version)
		//a.AppProfileV2.HelmRepoChartVersion = flags.Version
	}

	if len(flags.Set) > 0 {
		for _, set := range flags.Set {
			installParams = append(installParams, "--set", set)
		}
	}
	if flags.Values != "" {
		installParams = append(installParams, "-f", flags.Values)
	}
	installParams = append(installParams, "--timeout", "60m")
	installParams = append(installParams, commonParams...)

	fmt.Println("install helm application, this may take several minutes, please waiting...")

	_, err := tools.ExecCommand(nil, true, "helm", installParams...)
	if err != nil {
		return err
	}
	a.AppProfileV2.ReleaseName = releaseName
	a.AppProfileV2.ChartName = chartName
	a.SaveProfile()
	log.Infof(`helm nocalhost app installed, use "helm list -n %s" to get the information of the helm release`, a.GetNamespace())
	return nil
}

func (a *Application) installHelmInGit(flags *HelmFlags) error {

	resourcesPath := a.GetResourceDir()
	releaseName := a.Name

	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "-n", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	params := []string{"install", releaseName, resourcesPath[0]}
	if flags.Wait {
		params = append(params, "--wait")
	}
	if len(flags.Set) > 0 {
		for _, set := range flags.Set {
			params = append(params, "--set", set)
		}
	}
	if flags.Values != "" {
		params = append(params, "-f", flags.Values)
	}
	params = append(params, "--timeout", "60m")
	params = append(params, commonParams...)

	log.Info("building dependency...")
	depParams := []string{"dependency", "build", resourcesPath[0]}
	depParams = append(depParams, commonParams...)
	_, err := tools.ExecCommand(nil, true, "helm", depParams...)
	if err != nil {
		log.ErrorE(err, "fail to build dependency for helm app")
		return err
	}

	fmt.Println("install helm application, this may take several minutes, please waiting...")
	_, err = tools.ExecCommand(nil, true, "helm", params...)
	if err != nil {
		fmt.Printf("fail to install helm nocalhostApp, err:%v\n", err)
		return err
	}
	a.AppProfileV2.ReleaseName = releaseName
	a.SaveProfile()
	fmt.Printf(`helm application installed, use "helm list -n %s" to get the information of the helm release`+"\n", a.GetNamespace())
	return nil
}

func (a *Application) InstallDepConfigMap() error {
	appDep := a.GetDependencies()
	appEnv := a.GetInstallEnvForDep()
	if appDep != nil || len(appEnv.Global) > 0 || len(appEnv.Service) > 0 {
		var depForYaml = &struct {
			Dependency  []*SvcDependency  `json:"dependency" yaml:"dependency"`
			ReleaseName string            `json:"releaseName" yaml:"releaseName"`
			InstallEnv  *InstallEnvForDep `json:"env" yaml:"env"`
		}{
			Dependency: appDep,
			InstallEnv: appEnv,
		}
		// release name a.Name
		if a.AppProfileV2.AppType != Manifest {
			depForYaml.ReleaseName = a.Name
		}
		yamlBytes, err := yaml.Marshal(depForYaml)
		if err != nil {
			return errors.Wrap(err, "")
		}

		dataMap := make(map[string]string, 0)
		dataMap["nocalhost"] = string(yamlBytes)

		configMap := &corev1.ConfigMap{
			Data: dataMap,
		}

		var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
		rand.Seed(time.Now().UnixNano())
		b := make([]rune, 4)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		generateName := fmt.Sprintf("%s-%s", DependenceConfigMapPrefix, string(b))
		configMap.Name = generateName
		if configMap.Labels == nil {
			configMap.Labels = make(map[string]string, 0)
		}
		configMap.Labels["use-for"] = "nocalhost-dep"
		_, err = a.client.ClientSet.CoreV1().ConfigMaps(a.GetNamespace()).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			fmt.Errorf("fail to create dependency config %s\n", configMap.Name)
			return errors.Wrap(err, "")
		} else {
			a.AppProfileV2.DependencyConfigMapName = configMap.Name
			a.SaveProfile()
		}
	}
	log.Info("Dependency config map installed")
	return nil
}

func (a *Application) installManifestRecursively() error {
	//a.loadInstallManifest()
	log.Infof("%d manifest files to be installed", len(a.installManifest))
	if len(a.installManifest) > 0 {
		err := a.client.ApplyForCreate(a.installManifest, true, StandardNocalhostMetas(a.Name, a.GetNamespace()))
		if err != nil {
			return err
		}
	} else {
		log.Warn("nothing need to be installed ??")
	}
	return nil
}

func (a *Application) SetInstalledStatus(is bool) {
	a.AppProfileV2.Installed = is
	a.SaveProfile()
}

func (a *Application) loadInstallManifest() {
	a.installManifest = clientgoutils.
		LoadValidManifest(a.GetResourceDir(),
			append(a.getIgnoredPath(), a.getPreInstallFiles()...))
}

func (a *Application) loadPreInstallAndInstallManifest() {
	a.loadSortedPreInstallManifest()
	a.loadInstallManifest()
}

func (a *Application) loadUpgradePreInstallAndInstallManifest(resourcePath []string) {
	a.loadUpgradeSortedPreInstallManifest()
	a.loadUpgradeInstallManifest(resourcePath)
}

func (a *Application) loadUpgradeInstallManifest(upgradeResourcePath []string) {
	a.upgradeInstallManifest = clientgoutils.
		LoadValidManifest(a.getUpgradeResourceDir(upgradeResourcePath),
			append(a.getUpgradeIgnoredPath(), a.getUpgradePreInstallFiles()...))
}

func (a *Application) ignoredInUpgrade(manifest string) bool {
	for _, pre := range a.upgradeSortedPreInstallManifest {
		if pre == manifest {
			return true
		}
	}
	return false
}

func (a *Application) loadUpgradeSortedPreInstallManifest() {
	result := make([]string, 0)
	if a.AppProfileV2.PreInstall != nil {
		sort.Sort(ComparableItems(a.AppProfileV2.PreInstall))
		for _, item := range a.AppProfileV2.PreInstall {
			itemPath := filepath.Join(a.getUpgradeGitDir(), item.Path)
			if _, err2 := os.Stat(itemPath); err2 != nil {
				log.Warnf("%s is not a valid pre install manifest : %s\n", itemPath, err2.Error())
				continue
			}
			result = append(result, itemPath)
		}
	}
	a.upgradeSortedPreInstallManifest = result
}

func (a *Application) loadSortedPreInstallManifest() {
	result := make([]string, 0)
	if a.AppProfileV2.PreInstall != nil {
		sort.Sort(ComparableItems(a.AppProfileV2.PreInstall))
		for _, item := range a.AppProfileV2.PreInstall {
			itemPath := filepath.Join(a.getGitDir(), item.Path)
			if _, err2 := os.Stat(itemPath); err2 != nil {
				log.Warnf("%s is not a valid pre install manifest : %s\n", itemPath, err2.Error())
				continue
			}
			result = append(result, itemPath)
		}
	}
	a.sortedPreInstallManifest = result
}

func (a *Application) preInstall() {

	//a.loadSortedPreInstallManifest()

	if len(a.sortedPreInstallManifest) > 0 {
		log.Info("Run pre-install...")
		for _, item := range a.sortedPreInstallManifest {
			err := a.client.Create(item, true, false, StandardNocalhostMetas(a.Name, a.GetNamespace()))
			if err != nil {
				log.Warnf("error occurs when install %s : %s\n", item, err.Error())
			}
		}
	}
}
