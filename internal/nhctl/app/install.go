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
	"context"
	"fmt"
	"math/rand"
	"nocalhost/internal/nhctl/appmeta"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
)

// Install different type of Application: Helm, Manifest, Kustomize
func (a *Application) Install(flags *HelmFlags) (err error) {

	err = a.InstallDepConfigMap(a.appMeta)
	if err != nil {
		return errors.Wrap(err, "failed to install dep config map")
	}
	switch a.appMeta.ApplicationType {
	case appmeta.Helm, appmeta.HelmLocal:
		err = a.installHelm(a.appMeta, flags, a.ResourceTmpDir, false)
	case appmeta.HelmRepo:
		err = a.installHelm(a.appMeta, flags, a.ResourceTmpDir, true)
	case appmeta.Manifest, appmeta.ManifestLocal, appmeta.ManifestGit:
		err = a.InstallManifest(a.appMeta, a.ResourceTmpDir, true)
	case appmeta.KustomizeGit, appmeta.KustomizeLocal:
		err = a.InstallKustomize(a.appMeta, a.ResourceTmpDir, true)
	default:
		return errors.New(
			fmt.Sprintf(
				"unsupported application type, must be  %s, %s, %s, %s, %s, %s or %s",
				appmeta.HelmRepo, appmeta.Helm, appmeta.HelmLocal,
				appmeta.Manifest, appmeta.ManifestGit, appmeta.ManifestLocal, appmeta.KustomizeGit,
			),
		)
	}

	if err != nil {
		return err
	}

	a.appMeta.ApplicationState = appmeta.INSTALLED
	return a.appMeta.Update()
}

// Install different type of Application: Kustomize
func (a *Application) InstallKustomize(appMeta *appmeta.ApplicationMeta, resourceDir string, doApply bool) error {
	resourcesPath := a.GetResourceDir(resourceDir)
	if len(resourcesPath) > 1 {
		log.Warn(`There are multiple resourcesPath settings, will use first one`)
	}
	useResourcePath := resourcesPath[0]

	err := a.client.Apply(
		[]string{}, true,
		StandardNocalhostMetas(a.Name, a.NameSpace).
			SetDoApply(doApply).
			SetBeforeApply(
				func(manifest string) error {
					appMeta.Manifest = appMeta.Manifest + manifest
					return appMeta.Update()
				},
			),
		useResourcePath,
	)
	if err != nil {
		return err
	}
	return nil
}

// Install different type of Application: Manifest
func (a *Application) InstallManifest(appMeta *appmeta.ApplicationMeta, resourceDir string, doApply bool) error {
	p, err := a.GetProfile()
	if err != nil {
		return err
	}

	preInstallManifests, manifests := p.LoadManifests(resourceDir)

	err = a.client.ApplyAndWait(
		preInstallManifests, true,
		StandardNocalhostMetas(a.Name, a.NameSpace).
			SetDoApply(doApply).
			SetBeforeApply(
				func(manifest string) error {
					appMeta.PreInstallManifest = appMeta.PreInstallManifest + manifest
					return appMeta.Update()
				},
			),
	)
	if err != nil { // that's the error that could not be skip
		return err
	}

	return a.client.Apply(
		manifests, true,
		StandardNocalhostMetas(a.Name, a.NameSpace).
			SetDoApply(doApply).
			SetBeforeApply(
				func(manifest string) error {
					appMeta.Manifest = appMeta.Manifest + manifest
					return appMeta.Update()
				},
			),
		"",
	)
}

// Install different type of Application: Helm
func (a *Application) installHelm(
	appMeta *appmeta.ApplicationMeta, flags *HelmFlags, resourceDir string, fromRepo bool,
) error {
	log.Info("Updating helm repo...")
	_, err := tools.ExecCommand(nil, true, false, false, "helm", "repo", "update")
	if err != nil {
		log.Info(err.Error())
	}

	releaseName := a.Name
	appMeta.HelmReleaseName = releaseName
	if err := appMeta.Update(); err != nil {
		return err
	}

	commonParams := make([]string, 0)
	if a.NameSpace != "" {
		commonParams = append(commonParams, "--namespace", a.NameSpace)
	}
	if a.KubeConfig != "" {
		commonParams = append(commonParams, "--kubeconfig", a.KubeConfig)
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	var resourcesPath []string
	if !fromRepo {
		resourcesPath = a.GetResourceDir(resourceDir)
	}

	installParams := []string{"install", releaseName}
	if !fromRepo {
		installParams = append(installParams, resourcesPath[0])
		log.Info("building dependency...")
		depParams := []string{"dependency", "build", resourcesPath[0]}
		depParams = append(depParams, commonParams...)
		if _, err := tools.ExecCommand(nil, true, false, false, "helm", depParams...); err != nil {
			return errors.Wrap(err, "fail to build dependency for helm app")
		}
	} else {
		chartName := flags.Chart
		if a.appMeta.Config != nil && a.appMeta.Config.ApplicationConfig.Name != "" {
			chartName = a.appMeta.Config.ApplicationConfig.Name
		}
		if flags.RepoUrl != "" {
			installParams = append(installParams, chartName, "--repo", flags.RepoUrl)
		} else if flags.RepoName != "" {
			installParams = append(installParams, fmt.Sprintf("%s/%s", flags.RepoName, chartName))
		}

		if flags.Version != "" {
			installParams = append(installParams, "--version", flags.Version)
		} else {
			if a.appMeta.Config != nil && a.appMeta.Config.ApplicationConfig.HelmVersion != "" {
				installParams = append(installParams, "--version", a.appMeta.Config.ApplicationConfig.HelmVersion)
			}
		}
	}

	if flags.Wait {
		installParams = append(installParams, "--wait")
	}

	for _, set := range flags.Set {
		installParams = append(installParams, "--set", set)
	}

	if len(flags.Values) > 0 {
		for _, value := range flags.Values {
			installParams = append(installParams, "-f", value)
		}
	}
	installParams = append(installParams, "--timeout", "60m")
	installParams = append(installParams, commonParams...)

	log.Info("Installing helm application, this may take several minutes, please waiting...")

	if _, err := tools.ExecCommand(nil, true, false, false, "helm", installParams...); err != nil {
		return errors.Wrap(err, "fail to install helm application")
	}

	log.Infof(
		`helm nocalhost app installed, use "helm list -n %s" to
get the information of the helm release`, a.NameSpace,
	)
	return nil
}

func (a *Application) InstallDepConfigMap(appMeta *appmeta.ApplicationMeta) error {
	appDep := a.GetDependencies()
	appEnv := a.GetInstallEnvForDep()
	if len(appDep) > 0 || len(appEnv.Global) > 0 || len(appEnv.Service) > 0 {
		var depForYaml = &struct {
			Dependency  []*SvcDependency  `json:"dependency" yaml:"dependency"`
			ReleaseName string            `json:"releaseName" yaml:"releaseName"`
			InstallEnv  *InstallEnvForDep `json:"env" yaml:"env"`
		}{
			Dependency: appDep,
			InstallEnv: appEnv,
		}

		//if err := a.UpdateProfile(
		//	func(_ *profile.AppProfileV2) error {
		//		return nil
		//	},
		//); err != nil {
		//	return err
		//}

		// release name a.Name
		if a.appMeta.ApplicationType != appmeta.Manifest && a.appMeta.ApplicationType != appmeta.ManifestGit {
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

		appMeta.DepConfigName += generateName
		if err := appMeta.Update(); err != nil {
			return err
		}

		if _, err = a.client.ClientSet.CoreV1().ConfigMaps(a.NameSpace).Create(
			context.TODO(), configMap, metav1.CreateOptions{},
		); err != nil {
			return errors.Wrap(err, fmt.Sprintf("fail to create dependency config %s", configMap.Name))
		}
	}
	log.Logf("Dependency config map installed")
	return nil
}
