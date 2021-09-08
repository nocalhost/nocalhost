/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	kubeconfigCmd.AddCommand(kubeconfigRemoveCmd)
}

// Remove kubeconfig file
var kubeconfigRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove kubeconfig",
	Long:  `Remove kubeconfig`,
	Run: func(cmd *cobra.Command, args []string) {
		daemonClient, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
		if err != nil {
			log.FatalE(err, "")
		}
		// can set kubeConfig default value, but effect too much
		if len(kubeConfig) == 0 {
			kubeConfig = clientcmd.RecommendedHomeFile
		}
		if err := Prepare(); err != nil {
			return
		}
		if bytes, err := ioutil.ReadFile(kubeConfig); err == nil {
			if err = daemonClient.SendKubeconfigOperationCommand(bytes, nameSpace, command.OperationRemove); err != nil {
				log.Info(err)
			}
		}
	},
}
