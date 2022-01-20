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
	daemonInfoCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonCmd.AddCommand(daemonInfoCmd)
}

var daemonInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Get nhctl daemon info",
	Long:  `Get nhctl daemon info`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon_client.GetDaemonClient(isSudoUser)
		must(err)

		daemonServerInfo, err := client.SendGetDaemonServerInfoCommand()
		must(err)

		marshal, err := json.Marshal(daemonServerInfo)
		must(err)

		log.Infof("%s", marshal)
	},
}
