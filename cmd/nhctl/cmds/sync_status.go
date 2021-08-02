/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/syncthing/network/req"
)

var syncStatusOps = &app.SyncStatusOptions{}

func init() {
	//syncStatusCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	syncStatusCmd.Flags().StringVarP(&deployment, "deployment", "d", string(base.Deployment),
		"k8s deployment which your developing service exists",
	)
	syncStatusCmd.Flags().StringVarP(&serviceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.Override, "override", false,
		"override the remote changing according to the local sync folder",
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

		if !nhSvc.IsProcessor(){
			display(req.NotProcessor)
			return
		}

		client := nhSvc.NewSyncthingHttpClient()

		if syncStatusOps.Override {
			must(client.FolderOverride())
			display("Succeed")
			return
		}

		display(client.GetSyncthingStatus())
	},
}

func display(v interface{}) {
	marshal, _ := json.Marshal(v)
	fmt.Printf("%s", string(marshal))
}
