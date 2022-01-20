/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/daemon_client"
)

func init() {
	daemonStopCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonCmd.AddCommand(daemonStopCmd)
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop nhctl daemon",
	Long:  `Stop nhctl daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon_client.GetDaemonClient(isSudoUser)
		must(err)
		must(client.SendStopDaemonServerCommand())
	},
}
