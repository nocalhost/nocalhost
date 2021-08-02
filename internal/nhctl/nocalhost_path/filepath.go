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
	DefaultNhctlNameSpaceDirName = "ns"
)

func GetAppDbDir(ns, app string) string {
	return filepath.Join(GetAppDirUnderNs(app, ns), DefaultApplicationDbDir)
}

func GetAppDirUnderNs(appName string, namespace string) string {
	return filepath.Join(GetNhctlNameSpaceDir(), namespace, appName)
}

func GetNhctlHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName)
}

// .nh/nhctl/ns
func GetNhctlNameSpaceDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlNameSpaceDirName)
}
