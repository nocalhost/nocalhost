/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/cli-runtime/pkg/resource"
	flag "nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"os"
)

func (a *Application) PrepareForUpgrade(flags *flag.InstallFlags) error {

	var err error
	a.ResourceTmpDir, _ = ioutil.TempDir("", "")
	a.shouldClean = true
	if err = os.MkdirAll(a.ResourceTmpDir, DefaultNewFilePermission); err != nil {
		return errors.New("Fail to create tmp dir for upgrade")
	}
	if flags.GitUrl != "" {
		if err = downloadResourcesFromGit(flags.GitUrl, flags.GitRef, a.ResourceTmpDir); err != nil {
			return err
		}
	} else if flags.LocalPath != "" {
		a.ResourceTmpDir = flags.LocalPath
		a.shouldClean = false
	}

	if flags.OuterConfig == "" && a.GetType() == appmeta.HelmRepo {
		return nil
	}

	config, err := a.loadOrGenerateConfig(flags.OuterConfig, flags.Config, flags.ResourcePath, flags.AppType)
	if err != nil {
		return err
	}

	a.appMeta.Config = config
	a.appMeta.Config.Migrated = true
	return a.appMeta.Update()
}

func (a *Application) Upgrade(installFlags *flag.InstallFlags) error {

	switch a.GetType() {
	case appmeta.HelmRepo:

		if err := a.upgradeForHelm(installFlags, true); err != nil {
			return err
		}
	case appmeta.Helm, appmeta.HelmLocal:

		if err := a.upgradeForHelm(installFlags, false); err != nil {
			return err
		}
	case appmeta.Manifest, appmeta.ManifestLocal, appmeta.ManifestGit:

		log.Infof("dir: " + a.ResourceTmpDir)
		if err := a.PreUpgradeHook(); err != nil {
			return err
		}
		if err := a.upgradeForManifest(); err != nil {
			return err
		}
		if err := a.PostUpgradeHook(); err != nil {
			return err
		}
	case appmeta.KustomizeGit, appmeta.KustomizeLocal:

		if err := a.PreUpgradeHook(); err != nil {
			return err
		}
		if err := a.upgradeForKustomize(); err != nil {
			return err
		}
		if err := a.PostUpgradeHook(); err != nil {
			return err
		}
	default:
		return errors.New("Unsupported app type")
	}

	// prepare and store the delete hook while delete is trigger
	if err := a.PrepareForPreDeleteHook(); err != nil {
		return err
	}
	if err := a.PrepareForPostDeleteHook(); err != nil {
		return err
	}

	return a.CleanUpTmpResources()
}

func (a *Application) upgradeForKustomize() error {
	resourcesPath := a.GetResourceDir(a.ResourceTmpDir)
	if len(resourcesPath) > 1 {
		log.Warn(`There are multiple resourcesPath settings, will use first one`)
	}
	useResourcePath := resourcesPath[0]

	return a.client.Apply(
		[]string{}, true,
		StandardNocalhostMetas(a.Name, a.NameSpace).SetBeforeApply(
			func(manifest string) error {
				a.appMeta.Manifest = manifest
				return a.appMeta.Update()
			},
		),
		useResourcePath,
	)
}

func (a *Application) upgradeForManifest() error {
	manifests := a.GetAppMeta().GetApplicationConfig().LoadManifests(fp.NewFilePath(a.ResourceTmpDir))

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
	return a.appMeta.Update()
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
		err := clientgoutils.DeleteResourceInfo(info)
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

	_, err := tools.ExecCommand(nil, true, false, false, "helm", "repo", "update")
	if err != nil {
		log.Info(err.Error())
	}

	resourceDir := a.ResourceTmpDir
	appProfile, err := a.GetProfile()
	if err != nil {
		return err
	}

	releaseName := appProfile.ReleaseName
	if releaseName == "" {
		releaseName = a.appMeta.HelmReleaseName
	}
	if releaseName == "" {
		releaseName = a.appMeta.Application
	}

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
		resourcesPath := a.GetResourceDir(resourceDir)
		params = append(params, resourcesPath[0])
		log.Info("building dependency...")
		depParams := []string{"dependency", "build", resourcesPath[0]}
		depParams = append(depParams, commonParams...)
		if _, err = tools.ExecCommand(nil, true, false, false, "helm", depParams...); err != nil {
			return errors.Wrap(err, "fail to build dependency for helm app")
		}
	}

	if installFlags.HelmWait {
		params = append(params, "--wait")
	}
	if len(installFlags.HelmValueFile) > 0 {
		for _, values := range installFlags.HelmValueFile {
			params = append(params, "-f", values)
		}
	}
	for _, set := range installFlags.HelmSet {
		params = append(params, "--set", set)
	}
	params = append(params, "--timeout", "60m")
	params = append(params, commonParams...)

	log.Info("Upgrade helm application, this may take several minutes, please waiting...")

	_, err = tools.ExecCommand(nil, true, false, false, "helm", params...)
	return errors.Wrap(err, "")
}
