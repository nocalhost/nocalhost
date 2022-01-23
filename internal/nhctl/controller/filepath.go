/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/utils"
	"os"
	"path/filepath"
)

func (c *Controller) GetSyncThingPidFile() string {
	return filepath.Join(c.GetSyncDir(), _const.DefaultApplicationSyncPidFile)
}

func (c *Controller) GetSyncDir() string {
	dirPath := ""
	if c.Type == base.Deployment {
		dirPath = filepath.Join(c.getAppHomeDir(), _const.DefaultBinSyncThingDirName, c.Name)
	} else {
		dirPath = filepath.Join(c.getAppHomeDir(), _const.DefaultBinSyncThingDirName, string(c.Type)+"-"+c.Name)
	}
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		utils.Should(os.MkdirAll(dirPath, 0700))
	}
	return dirPath
}

func (c *Controller) getAppHomeDir() string {
	return nocalhost_path.GetAppDirUnderNs(c.AppName, c.NameSpace, c.AppMeta.NamespaceId)
}
