/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
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

// This path is independent from getConfigPathInGitResourcesDir()
func (a *Application) GetConfigPath() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigPath)
}

// Deprecated
func (a *Application) GetConfigV2Path() string {
	return filepath.Join(a.GetHomeDir(), DefaultApplicationConfigV2Path)
}

func (a *Application) GetHomeDir() string {
	return nocalhost_path.GetAppDirUnderNs(a.Name, a.NameSpace, a.appMeta.NamespaceId)
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
