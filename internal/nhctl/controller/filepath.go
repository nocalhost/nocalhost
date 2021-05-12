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

package controller

import (
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/utils"
	"os"
	"path/filepath"
)

// GetSyncThingPidFile Deprecated move to profile
func (c *Controller) GetSyncThingPidFile() string {
	return filepath.Join(c.GetApplicationSyncDir(), nocalhost.DefaultApplicationSyncPidFile)
}

func (c *Controller) GetApplicationSyncDir() string {
	dirPath := ""
	if c.Type == appmeta.Deployment {
		dirPath = filepath.Join(c.getAppHomeDir(), nocalhost.DefaultBinSyncThingDirName, c.Name)
	} else {
		dirPath = filepath.Join(c.getAppHomeDir(), nocalhost.DefaultBinSyncThingDirName, string(c.Type)+"-"+c.Name)
	}
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		utils.Should(os.MkdirAll(dirPath, 0700))
	}
	return dirPath
}

func (c *Controller) getAppHomeDir() string {
	return nocalhost_path.GetAppDirUnderNs(c.AppName, c.NameSpace)
}
