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

package svc

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"runtime"
	"strconv"
)

func (c *Controller) StopFileSyncOnly() error {
	var err error

	pf, err := c.GetPortForwardForSync()
	utils.Should(err)
	if pf != nil {
		utils.Should(c.EndDevPortForward(pf.LocalPort, pf.RemotePort))
	}

	// read and clean up pid file
	syncthingPid, syncThingPath, err := c.GetSyncThingPid()
	utils.ShouldI(err, "Failed to get background syncthing pid file")
	if syncthingPid != 0 {
		err = syncthing.Stop(syncthingPid, syncThingPath, "syncthing", true)
		if err != nil {
			if runtime.GOOS == "windows" {
				// in windows, it will raise a "Access is denied" err when killing progress, so we can ignore this err
				fmt.Printf("attempt to terminate syncthing process(pid: %d),"+
					" you can run `tasklist | findstr %d` to make sure process was exited\n",
					syncthingPid, syncthingPid)
			} else {
				log.WarnE(err, fmt.Sprintf("Failed to terminate syncthing process(pid: %d)", syncthingPid))
			}
		}
	}
	return err
}

func (c *Controller) GetSyncThingPid() (int, string, error) {
	pidFile := c.GetSyncThingPidFile()
	f, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, pidFile, err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, pidFile, err
	}
	return port, pidFile, nil
}

func (c *Controller) StopSyncAndPortForwardProcess(cleanRemoteSecret bool) error {
	err := c.StopFileSyncOnly()

	log.Info("Stopping port forward")
	utils.Should(c.StopAllPortForward())

	// Clean up secret
	if cleanRemoteSecret {
		svcProfile, _ := c.GetProfile()
		if svcProfile.SyncthingSecret != "" {
			log.Debugf("Cleaning up secret %s", svcProfile.SyncthingSecret)
			err = c.Client.DeleteSecret(svcProfile.SyncthingSecret)
			if err != nil {
				log.WarnE(err, "Failed to clean up syncthing secret")
			} else {
				svcProfile.SyncthingSecret = ""
			}
		}
	}

	return c.setSyncthingProfileEndStatus()
}

func (c *Controller) SetSyncingStatus(is bool) error {
	profileV2, err := profile.NewAppProfileV2ForUpdate(c.NameSpace, c.AppName)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.SvcProfileV2(c.Name, c.Type.String())
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.Syncing = is
	return profileV2.Save()
}
