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

func StartSyncthing(podName string, resume bool, stop bool, syncDouble *bool, override bool) {
	if !common.NocalhostSvc.IsInReplaceDevMode() && !common.NocalhostSvc.IsInDuplicateDevMode() {
		log.Fatalf("Service \"%s\" is not in developing", common.WorkloadName)
	}

	if !common.NocalhostSvc.IsProcessor() {
		log.Fatalf(
			"Service \"%s\" is not process by current device (DevMode is start by other device),"+
				" so can not operate the file sync", common.WorkloadName,
		)
	}

	// resume port-forward and syncthing
	if resume || stop {
		utils.ShouldI(common.NocalhostSvc.StopFileSyncOnly(), "Error occurs when stopping sync process")
		if stop {
			return
		}
	} else {
		if err := common.NocalhostSvc.FindOutSyncthingProcess(
			func(pid int) error {
				coloredoutput.Hint("Syncthing has been started")
				return errors.New("")
			},
		); err != nil {
			return
		}
	}

	svcProfile, _ := common.NocalhostSvc.GetProfile()
	if podName == "" {
		var err error
		if podName, err = common.NocalhostSvc.GetDevModePodName(); err != nil {
			must(err)
		}
	}
	log.Infof("Syncthing port-forward pod %s, namespace %s", podName, common.NocalhostApp.NameSpace)

	// Start a pf for syncthing
	must(common.NocalhostSvc.PortForward(podName, svcProfile.RemoteSyncthingPort, svcProfile.RemoteSyncthingPort, "SYNC"))

	str := strings.ReplaceAll(common.NocalhostSvc.GetSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")

	utils2.KillSyncthingProcess(str)

	if syncDouble == nil {
		flag := false

		config := common.NocalhostSvc.Config()
		if cfg := config.GetContainerDevConfig(DevStartOps.Container); cfg != nil && cfg.Sync != nil {
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
	dir := common.NocalhostSvc.GetSyncDir()
	if err2 := os.RemoveAll(dir); err2 != nil {
		log.Logf("Failed to delete dir: %s before starting syncthing, err: %v", dir, err2)
	}

	// TODO
	// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished),
	// the files that have been synchronized will not be synchronized.
	newSyncthing, err := common.NocalhostSvc.NewSyncthing(
		DevStartOps.Container, svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin, *syncDouble,
	)
	utils.ShouldI(err, "Failed to new syncthing")

	// try install syncthing
	var downloadVersion = daemon_common.Version

	// for debug only
	if DevStartOps.SyncthingVersion != "" {
		downloadVersion = DevStartOps.SyncthingVersion
	}

	_, err = syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, daemon_common.CommitId).InstallIfNeeded()
	mustI(
		err, "Failed to install syncthing, no syncthing available locally in "+
			newSyncthing.BinPath+" please try again.",
	)

	// starts up a local syncthing
	utils.ShouldI(newSyncthing.Run(context.TODO()), "Failed to run syncthing")

	must(common.NocalhostSvc.SetSyncingStatus(true))

	if override {
		var i = 10
		for {
			time.Sleep(time.Second)

			i--
			// to force override the remote changing
			client := common.NocalhostSvc.NewSyncthingHttpClient(2)
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
