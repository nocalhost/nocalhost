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
	"nocalhost/internal/nhctl/nocalhost"
	"path/filepath"
)

func (a *Application) getGitDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultResourcesDir)
}

func (a *Application) getUpgradeGitDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultUpgradeResourcesDir)
}

func (a *Application) getConfigPathInGitResourcesDir(configName string) string {
	if configName == "" {
		return filepath.Join(a.getGitDir(), DefaultApplicationConfigDirName, DefaultConfigNameInGitNocalhostDir)
	} else {
		return filepath.Join(a.getGitDir(), DefaultApplicationConfigDirName, configName)
	}
}

func (a *Application) GetPortSyncLogFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultSyncLogFileName)
}

func (a *Application) GetPortForwardLogFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultBackGroundPortForwardLogFileName)
}

func (a *Application) GetApplicationBackGroundOnlyPortForwardPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationOnlyPortForwardPidFile)
}

func (a *Application) GetFileLockPath(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), GetFileLockPath)
}

func (a *Application) GetApplicationSyncThingPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationSyncPidFile)
}

func (a *Application) GetApplicationOnlyPortForwardPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationOnlyPortForwardPidFile)
}

func (a *Application) GetApplicationBackGroundPortForwardPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationSyncPortForwardPidFile)
}

func (a *Application) getProfilePath() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationProfilePath)
}

func (a *Application) getProfileV2Path() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationProfileV2Path)
}

// This path is independent from getConfigPathInGitResourcesDir()
func (a *Application) GetConfigPath() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigPath)
}

func (a *Application) GetConfigV2Path() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigV2Path)
}

func (a *Application) GetHomeDir() string {
	return nocalhost.GetAppDir(a.Name)
}
