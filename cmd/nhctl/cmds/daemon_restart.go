/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	daemonRestartCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonCmd.AddCommand(daemonRestartCmd)
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart nhctl daemon",
	Long:  `Restart nhctl daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		isRunning := daemon_client.CheckIfDaemonServerRunning(isSudoUser)
		if isRunning {
			client, err := daemon_client.GetDaemonClient(isSudoUser)
			must(err)
			must(client.SendRestartDaemonServerCommand())
			log.Info("RestartDaemonServerCommand has been sent")
		} else {
			log.Warnf("Daemon Server(sudo:%t) is not running", isSudoUser)
			must(daemon_common.StartDaemonServerBySubProcess(isSudoUser))
		}
	},
}
