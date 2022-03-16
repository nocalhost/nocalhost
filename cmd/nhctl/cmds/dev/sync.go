/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package dev

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/coloredoutput"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"os"
	"strings"
	"time"
)

func (d *DevStartOps) StartSyncthing(podName string, resume bool, stop bool, syncDouble *bool, override bool) {
	if !d.NocalhostSvc.IsInReplaceDevMode() && !d.NocalhostSvc.IsInDuplicateDevMode() {
		log.Fatalf("Service \"%s\" is not in developing", common.WorkloadName)
	}

	if !d.NocalhostSvc.IsProcessor() {
		log.Fatalf(
			"Service \"%s\" is not process by current device (DevMode is start by other device),"+
				" so can not operate the file sync", common.WorkloadName,
		)
	}

	// resume port-forward and syncthing
	if resume || stop {
		utils.ShouldI(d.NocalhostSvc.StopFileSyncOnly(), "Error occurs when stopping sync process")
		if stop {
			return
		}
	} else {
		if err := d.NocalhostSvc.FindOutSyncthingProcess(
			func(pid int) error {
				coloredoutput.Hint("Syncthing has been started")
				return errors.New("")
			},
		); err != nil {
			return
		}
	}

	svcProfile, _ := d.NocalhostSvc.GetProfile()
	if podName == "" {
		var err error
		if podName, err = d.NocalhostSvc.GetDevModePodName(); err != nil {
			must(err)
		}
	}
	log.Infof("Syncthing port-forward pod %s, namespace %s", podName, d.NocalhostApp.NameSpace)

	// Start a pf for syncthing
	must(d.NocalhostSvc.PortForward(podName, svcProfile.RemoteSyncthingPort, svcProfile.RemoteSyncthingPort, "SYNC"))

	str := strings.ReplaceAll(d.NocalhostSvc.GetSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")

	utils2.KillSyncthingProcess(str)

	if syncDouble == nil {
		flag := false

		config := d.NocalhostSvc.Config()
		if cfg := config.GetContainerDevConfig(d.Container); cfg != nil && cfg.Sync != nil {
			switch cfg.Sync.Type {

			case _const.DefaultSyncType:
				flag = true

			default:
				flag = false
			}
		}

		syncDouble = &flag
	}

	// Delete service folder
	dir := d.NocalhostSvc.GetSyncDir()
	if err2 := os.RemoveAll(dir); err2 != nil {
		log.Logf("Failed to delete dir: %s before starting syncthing, err: %v", dir, err2)
	}

	// TODO
	// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished),
	// the files that have been synchronized will not be synchronized.
	newSyncthing, err := d.NocalhostSvc.NewSyncthing(
		d.Container, svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin, *syncDouble,
	)
	utils.ShouldI(err, "Failed to new syncthing")

	// try install syncthing
	var downloadVersion = daemon_common.Version

	// for debug only
	if d.SyncthingVersion != "" {
		downloadVersion = d.SyncthingVersion
	}

	_, err = syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, daemon_common.CommitId).InstallIfNeeded()
	mustI(
		err, "Failed to install syncthing, no syncthing available locally in "+
			newSyncthing.BinPath+" please try again.",
	)

	// starts up a local syncthing
	utils.ShouldI(newSyncthing.Run(context.TODO()), "Failed to run syncthing")

	must(d.NocalhostSvc.SetSyncingStatus(true))

	if override {
		var i = 10
		for {
			time.Sleep(time.Second)

			i--
			// to force override the remote changing
			client := d.NocalhostSvc.NewSyncthingHttpClient(2)
			err = client.FolderOverride()
			if err == nil {
				log.Info("Force overriding workDir's remote changing")
				break
			}

			if i < 0 {
				log.WarnE(err, "Fail to overriding workDir's remote changing")
				break
			}
		}
	}
}
