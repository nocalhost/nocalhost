package app

import (
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
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
