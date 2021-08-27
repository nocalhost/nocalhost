/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost_path

import (
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
)

const (
	DefaultApplicationDbDir      = "db"
	DefaultNhctlHomeDirName      = ".nh/nhctl"
	DefaultNocalhostHubDirName   = "nocalhost-hub"
	DefaultNhctlNameSpaceDirName = "ns"
	DefaultNhctlDevDirMappingDir     = "devmode/db"
	DefaultNhctlTestDevDirMappingDir = "testdevmode/db"
	DefaultNhctlKubeconfigDir        = "kubeconfig"
)

func GetNocalhostDevDirMapping() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlDevDirMappingDir)
}

func GetTestNocalhostDevDirMapping() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlTestDevDirMappingDir)
}

func GetAppDbDir(ns, app string) string {
	return filepath.Join(GetAppDirUnderNs(app, ns), DefaultApplicationDbDir)
}

func GetAppDirUnderNs(appName string, namespace string) string {
	return filepath.Join(GetNhctlNameSpaceDir(), namespace, appName)
}

func GetNhctlHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName)
}

func GetNhctlKubeconfigDir(name string) string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlKubeconfigDir, name)
}

// .nh/nhctl/ns
func GetNhctlNameSpaceDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlNameSpaceDirName)
}

func GetNocalhostHubDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNocalhostHubDirName)
}

func GetNocalhostIncubatorHubDir() string {
	return filepath.Join(GetNocalhostHubDir(), "incubator")
}

func GetNocalhostStableHubDir() string {
	return filepath.Join(GetNocalhostHubDir(), "stable")
}
