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
