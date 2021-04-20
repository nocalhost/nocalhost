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

package app

import (
	"fmt"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/utils"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"nocalhost/pkg/nhctl/log"
)

func (a *Application) StopAllPortForward(svcName string) error {
	appProfile, err := a.GetProfile()
	if err != nil {
		return err
	}
	svcProfile := appProfile.FetchSvcProfileV2FromProfile(svcName)

	for _, portForward := range svcProfile.DevPortForwardList {
		utils.Should(a.EndDevPortForward(svcName, portForward.LocalPort, portForward.RemotePort))
	}
	return nil
}

// port format 8080:80
func (a *Application) StopPortForwardByPort(svcName, port string) error {

	ports := strings.Split(port, ":")
	localPort, err := strconv.Atoi(ports[0])
	if err != nil {
		return errors.Wrap(err, "")
	}
	remotePort, err := strconv.Atoi(ports[1])
	if err != nil {
		return errors.Wrap(err, "")
	}
	return a.EndDevPortForward(svcName, localPort, remotePort)
}

func (a *Application) StopFileSyncOnly(svcName string) error {
	var err error

	pf, err := a.GetPortForwardForSync(svcName)
	utils.Should(err)
	if pf != nil {
		utils.Should(a.EndDevPortForward(svcName, pf.LocalPort, pf.RemotePort))
	}

	// Deprecated: port-forward has moved to daemon server
	portForwardPid, portForwardFilePath, err := a.GetBackgroundSyncPortForwardPid(svcName, false)
	utils.ShouldI(err, "Failed to get background port-forward pid file")
	if portForwardPid != 0 {
		utils.ShouldI(syncthing.Stop(portForwardPid, portForwardFilePath, "port-forward", true),
			fmt.Sprintf("Failed stop port-forward progress pid %d, please run `kill -9 %d`", portForwardPid, portForwardPid))
	}

	// read and clean up pid file
	syncthingPid, syncThingPath, err := a.GetBackgroundSyncThingPid(svcName, false)
	utils.ShouldI(err, "Failed to get background syncthing pid file")
	if syncthingPid != 0 {
		err = syncthing.Stop(syncthingPid, syncThingPath, "syncthing", true)
		if err != nil {
			if runtime.GOOS == "windows" {
				// in windows, it will raise a "Access is denied" err when killing progress, so we can ignore this err
				fmt.Printf("attempt to terminate syncthing process(pid: %d), you can run `tasklist | findstr %d` to make sure process was exited\n", portForwardPid, portForwardPid)
			} else {
				log.Warnf("Failed to terminate syncthing process(pid: %d), please run `kill -9 %d` manually, err: %s\n", portForwardPid, portForwardPid, err)
			}
		}
	}

	if err == nil { // none of them has error
		fmt.Printf("Background port-forward process: %d and  syncthing process: %d terminated.\n", portForwardPid, syncthingPid)
	}
	return err
}

func (a *Application) StopSyncAndPortForwardProcess(svcName string, cleanRemoteSecret bool) error {
	err := a.StopFileSyncOnly(svcName)

	log.Info("Stopping port forward")
	utils.Should(a.StopAllPortForward(svcName))

	// Clean up secret
	if cleanRemoteSecret {
		appProfile, _ := a.GetProfile()
		svcProfile := appProfile.FetchSvcProfileV2FromProfile(svcName)
		if svcProfile.SyncthingSecret != "" {
			log.Debugf("Cleaning up secret %s", svcProfile.SyncthingSecret)
			err = a.client.DeleteSecret(svcProfile.SyncthingSecret)
			if err != nil {
				log.WarnE(err, "Failed to clean up syncthing secret")
			} else {
				svcProfile.SyncthingSecret = ""
			}
		}
	}

	return a.SetSyncthingProfileEndStatus(svcName)
}

func (a *Application) DevEnd(svcName string, reset bool) error {
	if err := a.RollBack(svcName, reset); err != nil {
		if !reset {
			return err
		}
		log.WarnE(err, "something incorrect occurs when rolling back")
	}

	utils.ShouldI(a.appMeta.DeploymentDevEnd(svcName), "something incorrect occurs when updating secret")

	utils.ShouldI(a.StopSyncAndPortForwardProcess(svcName, true), "something incorrect occurs when stopping sync process")
	return nil
}
