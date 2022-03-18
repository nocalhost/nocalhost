/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	vpnCmd.AddCommand(vpnElevateCmd)
}

var vpnElevateCmd = &cobra.Command{
	Use:   "elevate",
	Short: "elevate",
	Long:  `elevate`,
	Run: func(cmd *cobra.Command, args []string) {
		// if not sudo and sudo daemon is not running, needs sudo permission
		if !util.IsAdmin() && !util.IsSudoDaemonServing() {
			if err := util.RunWithElevated(); err != nil {
				log.Fatal(err)
			}
		}
		if _, err := daemon_client.GetDaemonClient(true); err != nil {
			log.Fatal(err)
		}
	},
}
