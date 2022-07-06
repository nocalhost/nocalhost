/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

var (
	podName    string
	podSshPort int
	remotePort int
	localPort  int
)

func init() {
	sshRCmd.PersistentFlags().StringVarP(&podName, "pod", "", "", "pod name witch has ssh server")
	sshRCmd.PersistentFlags().IntVarP(&podSshPort, "sshport", "p", 50022, "pod ssh service port")
	sshRCmd.PersistentFlags().IntVarP(&remotePort, "remote", "", 2222, "remote port")
	sshRCmd.PersistentFlags().IntVarP(&localPort, "local", "", 2222, "local port")
	sshCmd.AddCommand(sshRCmd)
}

var sshRCmd = &cobra.Command{
	Use:     "reverse",
	Short:   "ssh reverse",
	Long:    `ssh reverse`,
	Example: `ssh reverse`,
	Run: func(cmd *cobra.Command, args []string) {

		port, _ := ports.GetAvailablePort()
		app := &app.Application{
			NameSpace:  common.NameSpace,
			KubeConfig: common.KubeConfig,
		}
		okchan := make(chan struct{})
		go func() {
			if err := app.PortForwardFollow(podName, port, podSshPort, okchan); err != nil {
				log.Fatal(err)
			}
		}()
		<-okchan
		err := utils.Reverse(
			utils.DefaultRoot,
			fmt.Sprintf("127.0.0.1:%d", port),
			fmt.Sprintf("0.0.0.0:%d", remotePort),
			fmt.Sprintf("127.0.0.1:%d", localPort),
		)
		if err != nil {
			log.Fatalf("error while create reverse tunnel, err: %v", err)
		}
	},
}
