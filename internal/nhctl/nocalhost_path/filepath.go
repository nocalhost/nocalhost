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
	DefaultApplicationDbDir          = "db"
	DefaultNhctlHomeDirName          = ".nh/nhctl"
	DefaultNocalhostHubDirName       = "nocalhost-hub"
	DefaultNhctlNameSpaceDirName     = "ns"
	DefaultNhctlDevDirMappingDir     = "devmode/db"
	DefaultNhctlTestDevDirMappingDir = "testdevmode/db"
	DefaultNhctlKubeconfigDir        = "kubeconfig"
	DefaultNhctlPortForward          = "portforward"
)

func GetNhctlHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName)
}

// .nh/nhctl/ns
func GetNhctlNameSpaceBaseDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlNameSpaceDirName)
}

func GetNocalhostDevDirMapping() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlDevDirMappingDir)
}

func GetTestNocalhostDevDirMapping() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlTestDevDirMappingDir)
}

func GetNidDir(namespace, nid string) string {
	return filepath.Join(GetNhctlNameSpaceBaseDir(), namespace, nid)
}

func GetAppDirUnderNs(appName, namespace, nid string) string {
	return filepath.Join(GetNhctlNameSpaceBaseDir(), namespace, nid, appName)
}

func GetAppDirUnderNsWithoutNid(appName string, namespace string) string {
	return filepath.Join(GetNhctlNameSpaceBaseDir(), namespace, appName)
}

func GetAppDbDir(ns, app, nid string) string {
	if nid != "" {
		return filepath.Join(GetAppDirUnderNs(app, ns, nid), DefaultApplicationDbDir)
	}
	return filepath.Join(GetAppDirUnderNsWithoutNid(app, ns), DefaultApplicationDbDir)
}

func GetAppDbDirWithoutNid(ns, app string) string {
	return filepath.Join(GetAppDirUnderNsWithoutNid(app, ns), DefaultApplicationDbDir)
}

func GetNhctlKubeconfigDir(name string) string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlKubeconfigDir, name)
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
