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

package appmeta

import (
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
)

const (
	ResourcePath ResourceType = "resourcePath"
	IgnorePath   ResourceType = "IgnorePath"

	PreInstall  ResourceType = "PreInstall"
	PostInstall ResourceType = "PostInstall"
)

type ResourceType string

func (a *ApplicationMeta) LoadAppInstallEnv(baseDir string) map[string]string {
	if a == nil || a.Config == nil || a.Config.ApplicationConfig == nil {
		return map[string]string{}
	}

	applicationCfg := a.Config.ApplicationConfig

	kvMap := map[string]string{}

	for _, f := range applicationCfg.EnvFrom.EnvFile {
		kvMap = fp.
			NewFilePath(baseDir).
			RelOrAbs(".nocalhost").
			RelOrAbs(f.Path).
			ReadAsEnvFileInto(kvMap)
	}

	// Env has a higher priority than envFrom
	for _, env := range applicationCfg.Env {
		kvMap[env.Name] = env.Value
	}
	return kvMap
}

func (a *ApplicationMeta) LoadPath(baseDir string, resourceType ResourceType) []string {
	if a == nil || a.Config == nil || a.Config.ApplicationConfig == nil {
		return nil
	}

	switch resourceType {

	case PreInstall:
		var paths profile.SortedRelPath
		paths = a.Config.ApplicationConfig.PreInstall
		return paths.Load(baseDir)

	case ResourcePath:
		var paths profile.RelPath
		paths = a.Config.ApplicationConfig.ResourcePath
		result := paths.Load(baseDir)
		if len(result) != 0 {
			return result
		}

		// if resource path not present, use whole dir as resource path
		return []string{baseDir}

	case IgnorePath:
		var paths profile.RelPath
		paths = a.Config.ApplicationConfig.IgnoredPath
		return paths.Load(baseDir)

	default:
		return make([]string, 0)
	}
}

func (a *ApplicationMeta) LoadManifestsEscapeHook(parentDir string) []string {
	preInstallManifests := a.LoadPath(parentDir, PreInstall)
	allManifests := a.LoadPath(parentDir, ResourcePath)
	ignores := a.LoadPath(parentDir, IgnorePath)

	return clientgoutils.LoadValidManifest(allManifests, append(preInstallManifests, ignores...))
}
