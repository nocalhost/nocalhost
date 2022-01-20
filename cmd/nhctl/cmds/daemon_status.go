/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	daemonStatusCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonCmd.AddCommand(daemonStatusCmd)
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get nhctl daemon status",
	Long:  `Get nhctl daemon status`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon_client.GetDaemonClient(isSudoUser)
		must(err)

		status, err := client.SendGetDaemonServerStatusCommand()
		must(err)

		marshal, err := json.Marshal(status)
		must(err)

		log.Infof("%s", marshal)
	},
}
