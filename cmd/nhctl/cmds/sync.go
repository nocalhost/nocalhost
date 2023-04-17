/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/cmd/nhctl/cmds/dev"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/model"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(
		&common.WorkloadName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	fileSyncCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
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

		nocalhostApp, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName,
			common.ServiceType)
		must(err)

		d := dev.DevStartOps{DevStartOptions: &model.DevStartOptions{}, NocalhostApp: nocalhostApp, NocalhostSvc: nocalhostSvc}
		d.StartSyncthing(
			"", fileSyncOps.Resume, fileSyncOps.Stop,
			// we do not read syncDouble from params now
			nil, fileSyncOps.Override,
		)
	},
}
