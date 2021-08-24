/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-ps"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"time"
)

var syncStatusOps = &app.SyncStatusOptions{}

func init() {
	//syncStatusCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	syncStatusCmd.Flags().StringVarP(
		&deployment, "deployment", "d", string(base.Deployment),
		"k8s deployment which your developing service exists",
	)
	syncStatusCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.Override, "override", false,
		"override the remote changing according to the local sync folder",
	)
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.WaitForSync, "wait", false,
		"wait for first sync process finished, default value is false",
	)
	syncStatusCmd.Flags().Int64Var(
		&syncStatusOps.Timeout, "timeout", 120,
		"wait for sync process finished timeout, default is 120 seconds, unit is seconds ",
	)
	rootCmd.AddCommand(syncStatusCmd)
}

var syncStatusCmd = &cobra.Command{
	Use:   "sync-status [NAME]",
	Short: "Files sync status",
	Long:  "Tracing the files sync status, include local folder and remote device",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		if err := initAppMutate(applicationName); err != nil {
			display(req.AppNotInstalledTemplate)
			return
		}

		nhSvc := initService(deployment, serviceType)

		if !nhSvc.IsInDevMode() {
			display(req.NotInDevModeTemplate)
			return
		}

		if !nhSvc.IsProcessor() {
			display(req.NotProcessor)
			return
		}

		// check if syncthing exists
		pid, err := nhSvc.GetSyncThingPid()
		if err != nil {
			display(req.NotSyncthingProcessFound)
			return
		}

		pro, err := ps.FindProcess(pid)
		if err != nil || pro == nil {
			display(req.NotSyncthingProcessFound)
			return
		}

		client := nhSvc.NewSyncthingHttpClient(2)

		if syncStatusOps.Override {
			must(client.FolderOverride())
			display("Succeed")
			return
		}

		if syncStatusOps.WaitForSync {
			waitForFirstSync(client, time.Second*time.Duration(syncStatusOps.Timeout))
			return
		}
		display(client.GetSyncthingStatus())
	},
}

func display(v interface{}) {
	marshal, _ := json.Marshal(v)
	fmt.Printf("%s", string(marshal))
}

func waitForFirstSync(client *req.SyncthingHttpClient, duration time.Duration) {
	timeout, cancelFunc := context.WithTimeout(context.Background(), duration)
	defer cancelFunc()

	for {
		select {
		case <-timeout.Done():
			display(
				req.SyncthingStatus{
					Status:    req.Error,
					Msg:       "wait for sync finished timeout",
					Tips:      "",
					OutOfSync: "",
				},
			)
			return
		default:
			time.Sleep(time.Millisecond * 100)
			events, err := client.EventsFolderCompletion()
			if err != nil || len(events) == 0 {
				continue
			}
			display(req.SyncthingStatus{Status: req.Idle, Msg: "sync finished", Tips: "", OutOfSync: ""})
			return
		}
	}
}
