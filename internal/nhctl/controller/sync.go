/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package controller

import (
	"fmt"
	"github.com/mitchellh/go-ps"
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

	pf, err := c.GetPortForwardForSync()
	utils.Should(err)
	if pf != nil {
		utils.Should(c.EndDevPortForward(pf.LocalPort, pf.RemotePort))
	}

	// read and clean up pid file
	syncthingPid, err := c.GetSyncThingPid()
	if err != nil {
		log.WarnE(err, "Failed to get syncthing pid")
		return nil
	}
	if syncthingPid != 0 {
		if err = syncthing.Stop(syncthingPid, true); err != nil {
			if runtime.GOOS == "windows" {
				// in windows, it will raise a "Access is denied" err when killing progress, so we can ignore this err
				fmt.Printf(
					"attempt to terminate syncthing process(pid: %d),"+
						" you can run `tasklist | findstr %d` to make sure process was exited\n",
					syncthingPid, syncthingPid,
				)
			} else {
				log.WarnE(err, fmt.Sprintf("Failed to terminate syncthing process(pid: %d)", syncthingPid))
			}
			return err
		}
	}
	return err
}

func (c *Controller) FindOutSyncthingProcess(whileProcessFound func(int) error) error {
	previousSyncThingPid, err := c.GetSyncThingPid()

	if err != nil {
		log.Info("No previous syncthing pid found(ignore)")
		log.LogE(err)
	} else {
		pro, err := ps.FindProcess(previousSyncThingPid)
		if err == nil && pro == nil {
			log.Infof("No previous syncthing process (%d) found", previousSyncThingPid)
		} else {
			return whileProcessFound(previousSyncThingPid)
		}
	}
	return nil
}

func (c *Controller) GetSyncThingPid() (int, error) {
	pidFile := c.GetSyncThingPidFile()
	f, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, errors.Wrap(err, "")
	}
	return port, nil
}

func (c *Controller) StopSyncAndPortForwardProcess(cleanRemoteSecret bool) error {
	err := c.StopFileSyncOnly()
	if err != nil {
		return err
	}

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
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to get controller profile")
			}

			svcProfile.Syncing = is
			return nil
		},
	)
}
