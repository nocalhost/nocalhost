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
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"os"
	"path/filepath"
)

func (a *Application) getDbDir() string {
	return filepath.Join(a.GetHomeDir(), nocalhost_path.DefaultApplicationDbDir)
}

// deprecated
func (a *Application) getResourceDir() string {
	return filepath.Join(a.GetHomeDir(), "resources")
}

func (a *Application) getGitNocalhostDir() string {
	return filepath.Join(a.ResourceTmpDir, DefaultGitNocalhostDir)
}

func (a *Application) getConfigPathInGitResourcesDir(configName string) string {
	if configName == "" {
		return filepath.Join(a.ResourceTmpDir, DefaultGitNocalhostDir, DefaultConfigNameInGitNocalhostDir)
	}
	return filepath.Join(a.ResourceTmpDir, DefaultGitNocalhostDir, configName)
}

func (a *Application) getProfilePath() string {
	return filepath.Join(a.GetHomeDir(), nocalhost.DefaultApplicationProfilePath)
}

// Deprecated
func (a *Application) getProfileV2Path() string {
	return filepath.Join(a.GetHomeDir(), nocalhost.DefaultApplicationProfileV2Path)
}

// This path is independent from getConfigPathInGitResourcesDir()
func (a *Application) GetConfigPath() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigPath)
}

// Deprecated
func (a *Application) GetConfigV2Path() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigV2Path)
}

func (a *Application) GetHomeDir() string {
	return nocalhost_path.GetAppDirUnderNs(a.Name, a.NameSpace)
}

func (a *Application) getSecretGeneratedSignDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultSecretGenSign)
}

func (a *Application) MarkAsGenerated() {
	_, _ = os.Create(a.getSecretGeneratedSignDir())
}

func (a *Application) HasBeenGenerateSecret() bool {
	_, err := os.Stat(a.getSecretGeneratedSignDir())
	return err == nil
}
