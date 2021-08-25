/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) PreInstallHook() error {
	return a.applyManifestAndWaitCompleteThen(
		a.GetAppMeta().GetApplicationConfig().PreInstall,
		func(manifest string) error {
			if manifest != "" {
				log.Info("Executing pre-install hook")
			}
			a.GetAppMeta().PreInstallManifest = a.GetAppMeta().PreInstallManifest + manifest
			return a.GetAppMeta().Update()
		}, true,
	)
}

func (a *Application) PostInstallHook() error {

	return a.applyManifestAndWaitCompleteThen(
		a.GetAppMeta().GetApplicationConfig().PostInstall,
		func(manifest string) error {
			if manifest != "" {
				log.Info("Executing post-install hook")
			}
			a.GetAppMeta().PostInstallManifest = a.GetAppMeta().PostInstallManifest + manifest
			return a.GetAppMeta().Update()
		}, true,
	)
}

func (a *Application) PreUpgradeHook() error {
	return a.applyManifestAndWaitCompleteThen(
		a.GetAppMeta().GetApplicationConfig().PreUpgrade,
		func(manifest string) error {
			if manifest != "" {
				log.Info("Executing pre-upgrade hook")
			}
			a.GetAppMeta().PreUpgradeManifest = a.GetAppMeta().PreUpgradeManifest + manifest
			return a.GetAppMeta().Update()
		}, true,
	)
}

func (a *Application) PostUpgradeHook() error {
	return a.applyManifestAndWaitCompleteThen(
		a.GetAppMeta().GetApplicationConfig().PostUpgrade,
		func(manifest string) error {
			if manifest != "" {
				log.Info("Executing post-upgrade hook")
			}
			a.GetAppMeta().PostUpgradeManifest = a.GetAppMeta().PostUpgradeManifest + manifest
			return a.GetAppMeta().Update()
		}, true,
	)
}

func (a *Application) PrepareForPreDeleteHook() error {
	return a.applyManifestAndWaitCompleteThen(
		a.GetAppMeta().GetApplicationConfig().PreDelete,
		func(manifest string) error {
			a.GetAppMeta().PreDeleteManifest = manifest
			return a.GetAppMeta().Update()
		}, false,
	)
}

func (a *Application) PrepareForPostDeleteHook() error {
	return a.applyManifestAndWaitCompleteThen(
		a.GetAppMeta().GetApplicationConfig().PostDelete,
		func(manifest string) error {
			a.GetAppMeta().PostDeleteManifest = manifest
			return a.GetAppMeta().Update()
		}, false,
	)
}

func (a *Application) applyManifestAndWaitCompleteThen(weightablePath []*profile.WeightablePath, beforeApplyManifest func(string) error, doApply bool) error {
	var path profile.SortedRelPath = weightablePath
	return a.client.ApplyAndWait(
		path.Load(fp.NewFilePath(a.ResourceTmpDir)), true,
		StandardNocalhostMetas(a.Name, a.NameSpace).
			SetDoApply(doApply).
			SetBeforeApply(beforeApplyManifest),
	)
}
