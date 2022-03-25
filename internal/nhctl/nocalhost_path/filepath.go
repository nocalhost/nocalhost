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
	DefaultApplicationDbDir   = "db"
	DefaultNhctlHomeDirName   = ".nh/nhctl"
	DefaultNhctlKubeconfigDir = "kubeconfig"
	DefaultNhctlPortForward   = "portforward"
)

var (
	NhctlHome              = filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName) // ~/.nh/nhctl
	NhctlNsDir             = filepath.Join(NhctlHome, "ns")                              // ~/.nh/nhctl/ns
	NhctlDevDirMapping     = filepath.Join(NhctlHome, "devmode/db")                      //  ~/.nh/nhctl/devmode/db
	NhctlTestDevDirMapping = filepath.Join(NhctlHome, "testdevmode/db")
	NhctlHubDir            = filepath.Join(NhctlHome, "nocalhost-hub")
	NhctlIncubatorHubDir   = filepath.Join(NhctlHubDir, "incubator")
	NhctlStableHubDir      = filepath.Join(NhctlHubDir, "stable")
)

func GetNidDir(namespace, nid string) string {
	return filepath.Join(NhctlNsDir, namespace, nid)
}

func GetAppDirUnderNs(appName, namespace, nid string) string {
	return filepath.Join(NhctlNsDir, namespace, nid, appName)
}

func GetAppDirUnderNsWithoutNid(appName string, namespace string) string {
	return filepath.Join(NhctlNsDir, namespace, appName)
}

func GetAppDbDir(ns, app, nid string) string {
	if nid != "" {
		return filepath.Join(GetAppDirUnderNs(app, ns, nid), DefaultApplicationDbDir)
	}
	return filepath.Join(GetAppDirUnderNsWithoutNid(app, ns), DefaultApplicationDbDir)
}

func GetNhctlKubeconfigDir(name string) string {
	return filepath.Join(NhctlHome, DefaultNhctlKubeconfigDir, name)
}
