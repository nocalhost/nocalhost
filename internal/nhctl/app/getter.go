/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package app

import (
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/clientgoutils"
)

func (a *Application) GetType() appmeta.AppType {
	return a.appMeta.ApplicationType
}

func (a *Application) IsHelm() bool {
	t := a.GetType()
	return t == appmeta.Helm || t == appmeta.HelmRepo || t == appmeta.HelmLocal
}

func (a *Application) IsManifest() bool {
	t := a.GetType()
	return t == appmeta.Manifest || t == appmeta.ManifestLocal || t == appmeta.ManifestGit
}

func (a *Application) IsKustomize() bool {
	t := a.GetType()
	return t == appmeta.KustomizeGit
}

func (a *Application) GetClient() *clientgoutils.ClientGoUtils {
	return a.client
}
