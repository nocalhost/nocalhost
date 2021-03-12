/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"context"
	"fmt"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/syncthing/terminate"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"nocalhost/pkg/nhctl/log"
)

func (a *Application) StopAllPortForward(svcName string) error {

	for _, portForward := range a.GetSvcProfileV2(svcName).DevPortForwardList {
		log.Infof("Stopping process %d", portForward.Pid)
		err := terminate.Terminate(portForward.Pid, true, "port-forward")
		if err != nil {
			log.WarnE(err, "")
		}
	}

	// Clean up port-forward status
	a.GetSvcProfileV2(svcName).DevPortForwardList = make([]*profile.DevPortForward, 0)
	return a.SaveProfile()
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
	a.EndDevPortForward(svcName, localPort, remotePort)

	return nil
}

func (a *Application) StopSyncAndPortForwardProcess(svcName string) error {
	var err error
	fileSyncOps := &FileSyncOptions{}
	devStartOptions := &DevStartOptions{}

	newSyncthing, err := a.NewSyncthing(svcName, "", devStartOptions.LocalSyncDir, fileSyncOps.SyncDouble)
	if err != nil {
		log.Warnf("Failed to start syncthing process: %s", err.Error())
		return err
	}

	// read and clean up pid file
	portForwardPid, portForwardFilePath, err := a.GetBackgroundSyncPortForwardPid(svcName, false)
	if err != nil {
		log.Warn("Failed to get background port-forward pid file, ignored")
	}
	if portForwardPid != 0 {
		err = newSyncthing.Stop(portForwardPid, portForwardFilePath, "port-forward", true)
		if err != nil {
			log.Warnf("Failed stop port-forward progress pid %d, please run `kill -9 %d` by manual, err: %s\n", portForwardPid, portForwardPid, err)
		}
	}

	// read and clean up pid file
	syncthingPid, syncThingPath, err := a.GetBackgroundSyncThingPid(svcName, false)
	if err != nil {
		log.Warn("Failed to get background syncthing pid file, ignored")
	}
	if syncthingPid != 0 {
		err = newSyncthing.Stop(syncthingPid, syncThingPath, "syncthing", true)
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

	log.Info("Stopping port forward")
	err = a.StopAllPortForward(svcName)
	if err != nil {
		log.WarnE(err, err.Error())
	}

	// Clean up secret
	svcProfile := a.GetSvcProfileV2(svcName)
	if svcProfile.SyncthingSecret != "" {
		log.Debugf("Cleaning up secret %s", svcProfile.SyncthingSecret)
		err = a.client.DeleteSecret(svcProfile.SyncthingSecret)
		if err != nil {
			log.WarnE(err, "Failed to clean up syncthing secret")
		} else {
			svcProfile.SyncthingSecret = ""
		}
	}

	// set profile status
	// set port-forward port and ignore result
	// err = a.SetSyncthingPort(svcName, 0, 0, 0, 0)
	err = a.SetSyncthingProfileEndStatus(svcName)
	return err
}

func (a *Application) Reset(svcName string) {
	var err error
	err = a.StopSyncAndPortForwardProcess(svcName)
	if err != nil {
		log.Warnf("something incorrect occurs when stopping sync process: %s", err.Error())
	}
	err = a.RollBack(context.TODO(), svcName, true)
	if err != nil {
		log.Warnf("something incorrect occurs when rolling back: %s", err.Error())
	}
	err = a.SetDevEndProfileStatus(svcName)
	if err != nil {
		log.Warnf("fail to update \"developing\" status")
	}
}

func (a *Application) EndDevelopMode(svcName string) error {
	var err error

	log.Info("Ending devMode...")

	// end file sync
	//log.Info("Terminating file sync process...")
	//err = a.stopSyncAndPortForwardProcess(svcName)
	//if err != nil {
	//	log.WarnE(err, "Error occurs when stopping sync process")
	//	return err
	//}

	// roll back workload
	log.Debug("Rolling back workload...")
	err = a.RollBack(context.TODO(), svcName, false)
	if err != nil {
		log.Error("Failed to rollback")
		return err
	}

	err = a.SetDevEndProfileStatus(svcName)
	if err != nil {
		log.Warn("Failed to update \"developing\" status")
		return err
	}
	return nil
}
