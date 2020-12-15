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
	"path/filepath"

	"nocalhost/internal/nhctl/utils"
)

func (a *Application) getGitDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultResourcesDir)
}

func (a *Application) getConfigPathInGitResourcesDir() string {
	return filepath.Join(a.getGitDir(), DefaultApplicationConfigDirName, DefaultApplicationConfigName)
}

func (a *Application) GetSyncThingBinDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName, DefaultBinDirName, DefaultBinSyncThingDirName)
}

func (a *Application) GetLogDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName, DefaultLogDirName)
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

func (a *Application) GetApplicationSyncThingPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationSyncPidFile)
}

func (a *Application) GetApplicationOnlyPortForwardPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationOnlyPortForwardPidFile)
}

func (a *Application) GetConfigDir() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigDirName)
}

func (a *Application) GetConfigPath() string {
	return filepath.Join(a.GetConfigDir(), DefaultApplicationConfigName)
}

func (a *Application) GetApplicationBackGroundPortForwardPidFile(deployment string) string {
	return filepath.Join(a.GetApplicationSyncDir(deployment), DefaultApplicationSyncPortForwardPidFile)
}

func (a *Application) getProfilePath() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationProfilePath)
}

func (a *Application) GetHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName, DefaultApplicationDirName, a.Name)
}
