/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	kubeconfigCmd.AddCommand(kubeconfigAddCmd)
}

// Add kubeconfig
var kubeconfigAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add kubeconfig",
	Long:  `Add kubeconfig`,
	Run: func(cmd *cobra.Command, args []string) {
		daemonClient, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
		if err != nil {
			log.FatalE(err, "")
		}
		if err := common.Prepare(); err != nil {
			return
		}
		if bytes, err := ioutil.ReadFile(common.KubeConfig); err == nil {
			if err = daemonClient.SendKubeconfigOperationCommand(bytes, common.NameSpace, command.OperationAdd); err != nil {
				log.Info(err)
			}
		}
	},
}
