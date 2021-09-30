/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	utils2 "nocalhost/pkg/nhctl/utils"
	"os"
	"strings"
	"time"
)

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(
		&deployment, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	fileSyncCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	fileSyncCmd.Flags().BoolVarP(
		&fileSyncOps.SyncDouble, "double", "b", false,
		"if use double side sync",
	)
	fileSyncCmd.Flags().BoolVar(
		&fileSyncOps.Resume, "resume", false,
		"resume file sync",
	)
	fileSyncCmd.Flags().BoolVar(&fileSyncOps.Stop, "stop", false, "stop file sync")
	fileSyncCmd.Flags().StringSliceVarP(
		&fileSyncOps.SyncedPattern, "synced-pattern", "s", []string{},
		"local synced pattern",
	)
	fileSyncCmd.Flags().StringSliceVarP(
		&fileSyncOps.IgnoredPattern, "ignored-pattern", "i", []string{},
		"local ignored pattern",
	)
	fileSyncCmd.Flags().StringVar(&fileSyncOps.Container, "container", "", "container name of pod to sync")
	fileSyncCmd.Flags().BoolVar(
		&fileSyncOps.Override, "overwrite", true,
		"override the remote changing according to the local sync folder while start up",
	)
	rootCmd.AddCommand(fileSyncCmd)
}

var fileSyncCmd = &cobra.Command{
	Use:   "sync [NAME]",
	Short: "Sync files to remote Pod in Kubernetes",
	Long:  `Sync files to remote Pod in Kubernetes`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]

		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		StartSyncthing(
			"", fileSyncOps.Resume, fileSyncOps.Stop, fileSyncOps.Container,
			&fileSyncOps.SyncDouble, fileSyncOps.Override,
		)
	},
}

func StartSyncthing(podName string, resume bool, stop bool, container string, syncDouble *bool, override bool) {
	if !nocalhostSvc.IsInDevMode() {
		log.Fatalf("Service \"%s\" is not in developing", deployment)
	}

	if !nocalhostSvc.IsProcessor() {
		log.Fatalf(
			"Service \"%s\" is not process by current device (DevMode is start by other device),"+
				" so can not operate the file sync", deployment,
		)
	}

	// resume port-forward and syncthing
	if resume || stop {
		utils.ShouldI(nocalhostSvc.StopFileSyncOnly(), "Error occurs when stopping sync process")
		if stop {
			return
		}
	} else {
		if err := nocalhostSvc.FindOutSyncthingProcess(
			func(pid int) error {
				coloredoutput.Hint("Syncthing has been started")
				return errors.New("")
			},
		); err != nil {
			return
		}
	}

	if podName == "" {
		var err error
		if podName, err = nocalhostSvc.BuildPodController().GetNocalhostDevContainerPod(); err != nil {
			must(err)
		}
	}
	log.Infof("Syncthing port-forward pod %s, namespace %s", podName, nocalhostApp.NameSpace)

	svcProfile, _ := nocalhostSvc.GetProfile()
	// Start a pf for syncthing
	must(nocalhostSvc.PortForward(podName, svcProfile.RemoteSyncthingPort, svcProfile.RemoteSyncthingPort, "SYNC"))

	str := strings.ReplaceAll(nocalhostSvc.GetApplicationSyncDir(), nocalhost_path.GetNhctlHomeDir(), "")
	
	utils2.KillSyncthingProcess(str)

	if syncDouble == nil {
		flag := false

		if config, err := nocalhostSvc.GetConfig(); err == nil {
			if cfg := config.GetContainerDevConfig(container); cfg != nil && cfg.Sync != nil {
				switch cfg.Sync.Type {

				case syncthing.DefaultSyncMode:
					flag = true

				default:
					flag = false
				}
			}
		}

		syncDouble = &flag
	}

	// Delete service folder
	dir := nocalhostSvc.GetApplicationSyncDir()
	if err2 := os.RemoveAll(dir); err2 != nil {
		log.Logf("Failed to delete dir: %s before starting syncthing, err: %v", dir, err2)
	}

	// TODO
	// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished),
	// the files that have been synchronized will not be synchronized.
	newSyncthing, err := nocalhostSvc.NewSyncthing(
		container, svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin, *syncDouble,
	)
	utils.ShouldI(err, "Failed to new syncthing")

	// try install syncthing
	var downloadVersion = Version

	// for debug only
	if devStartOps.SyncthingVersion != "" {
		downloadVersion = devStartOps.SyncthingVersion
	}

	_, err = syncthing.NewInstaller(newSyncthing.BinPath, downloadVersion, GitCommit).InstallIfNeeded()
	mustI(
		err, "Failed to install syncthing, no syncthing available locally in "+
			newSyncthing.BinPath+" please try again.",
	)

	// starts up a local syncthing
	utils.ShouldI(newSyncthing.Run(context.TODO()), "Failed to run syncthing")

	must(nocalhostSvc.SetSyncingStatus(true))

	if override {
		var i = 10
		for {
			time.Sleep(time.Second)

			i--
			// to force override the remote changing
			client := nocalhostSvc.NewSyncthingHttpClient(2)
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
