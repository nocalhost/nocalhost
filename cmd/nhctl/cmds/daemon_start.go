/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/daemon_common"
	"nocalhost/internal/nhctl/daemon_server"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	daemonStartCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonStartCmd.Flags().BoolVarP(&runInBackground, "daemon", "d", false, "Is run as daemon(background)")
	daemonCmd.AddCommand(daemonStartCmd)
}

// This command is run by daemon client as a background progress usually
var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start nhctl daemon",
	Long:  `Start nhctl daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		log.AddField("APP", "daemon-server")
		if runInBackground {
			must(daemon_common.StartDaemonServerBySubProcess(isSudoUser))
			return
		}

		must(daemon_server.StartDaemon(isSudoUser, Version, GitCommit))
	},
}
