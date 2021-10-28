/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/fp"
	"regexp"
	"strings"
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

	if err := a.InstallDepConfigMap(a.appMeta); err != nil {
		return errors.Wrap(err, "failed to install dep config map")
	}

	switch a.appMeta.ApplicationType {
	case appmeta.Helm, appmeta.HelmLocal:
		err = a.installHelm(flags, false)
	case appmeta.HelmRepo:
		err = a.installHelm(flags, true)
	case appmeta.Manifest, appmeta.ManifestLocal, appmeta.ManifestGit:
		if err := a.PreInstallHook(); err != nil {
			return err
		}
		if err := a.InstallManifest(true); err != nil {
			return err
		}
		if err := a.PostInstallHook(); err != nil {
			return err
		}
	case appmeta.KustomizeGit, appmeta.KustomizeLocal:
		if err := a.PreInstallHook(); err != nil {
			return err
		}
		if err := a.InstallKustomize(true); err != nil {
			return err
		}
		if err := a.PostInstallHook(); err != nil {
			return err
		}
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

	// prepare and store the delete hook while delete is trigger
	if err := a.PrepareForPreDeleteHook(); err != nil {
		return err
	}

	if err := a.PrepareForPostDeleteHook(); err != nil {
		return err
	}

	a.appMeta.ApplicationState = appmeta.INSTALLED

	if err := a.appMeta.Update(); err != nil {
		return err
	}

	return a.CleanUpTmpResources()
}

// Install different type of Application: Kustomize
func (a *Application) InstallKustomize(doApply bool) error {
	resourcesPath := a.GetResourceDir(a.ResourceTmpDir)
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
					a.GetAppMeta().Manifest = a.GetAppMeta().Manifest + manifest
					return a.GetAppMeta().Update()
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
func (a *Application) InstallManifest(doApply bool) error {
	manifestPaths := a.GetAppMeta().GetApplicationConfig().LoadManifests(fp.NewFilePath(a.ResourceTmpDir))

	return a.client.Apply(
		manifestPaths, true,
		StandardNocalhostMetas(a.Name, a.NameSpace).
			SetDoApply(doApply).
			SetBeforeApply(
				func(manifest string) error {
					a.GetAppMeta().Manifest = a.GetAppMeta().Manifest + manifest
					return a.GetAppMeta().Update()
				},
			),
		"",
	)
}

// Install different type of Application: Helm
func (a *Application) installHelm(flags *HelmFlags, fromRepo bool) error {
	log.Info("Updating helm repo...")
	_, err := tools.ExecCommand(nil, true, false, false, "helm", "repo", "update")
	if err != nil {
		log.Info(err.Error())
	}

	releaseName := a.Name
	a.GetAppMeta().HelmReleaseName = releaseName
	if err = a.GetAppMeta().Update(); err != nil {
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

	var (
		resourcesPath []string
		installParams = []string{"install", releaseName}
	)

	if !fromRepo {
		resourcesPath = a.GetResourceDir(a.ResourceTmpDir)
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
		if len(findRepoNameFromLocal(flags.RepoUrl)) != 0 {
			installParams = append(installParams, fmt.Sprintf("%s/%s", findRepoNameFromLocal(flags.RepoUrl), chartName))
		} else if flags.RepoUrl != "" {
			installParams = append(installParams, chartName, "--repo", flags.RepoUrl)
		} else if flags.RepoName != "" {
			installParams = append(installParams, fmt.Sprintf("%s/%s", flags.RepoName, chartName))
		}

		if withCredential, _ := regexp.MatchString(`//(?P<username>.*?):(?P<password>[a-zA-Z0-9+]+)@`, flags.RepoUrl); withCredential {
			if compile, err := regexp.Compile(`//(?P<username>.*?):(?P<password>[a-zA-Z0-9+]+)@`); err == nil {
				if submatch := compile.FindStringSubmatch(flags.RepoUrl); len(submatch) == 3 {
					installParams = append(installParams, "--username", submatch[1])
					installParams = append(installParams, "--password", submatch[2])
				}
			}
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

// judge helm repo already exist or not, if exist, then can install app by using repo name, otherwise using repo url
func findRepoNameFromLocal(helmRepoUrl string) (repoName string) {
	// it's a private repo but provide username and password already
	if withCredential, _ := regexp.MatchString("(.*?):(.*?)@(.*?)", helmRepoUrl); withCredential {
		return
	}

	// remove https:// https:// header
	if strings.Index(helmRepoUrl, "//") >= 0 {
		helmRepoUrl = helmRepoUrl[strings.Index(helmRepoUrl, "//")+2:]
	}
	if output, err := tools.ExecCommand(nil, false, false, true, "helm", "repo", "list", "--output", "json"); err == nil {
		var repoList []RepoDto
		if err = json.Unmarshal([]byte(output), &repoList); err == nil {
			for _, dto := range repoList {
				if strings.Contains(dto.Url, helmRepoUrl) {
					return dto.Name
				}
			}
		}
	}
	return
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

type RepoDto struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}
