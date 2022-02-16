/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/driver"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	reconnectCmd.Flags().StringVar(&common.KubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	reconnectCmd.Flags().StringVarP(&common.NameSpace, "namespace", "n", "", "namespace")
	reconnectCmd.Flags().StringVar(&workloads, "workloads", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	reconnectCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	vpnCmd.AddCommand(reconnectCmd)
}

var reconnectCmd = &cobra.Command{
	Use:   "reconnect",
	Short: "reconnect",
	Long:  `reconnect`,
	PreRun: func(*cobra.Command, []string) {
		util.InitLogger(util.Debug)
		if util.IsWindows() {
			_ = driver.InstallWireGuardTunDriver()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// if not sudo and sudo daemon is not running, needs sudo permission
		if !util.IsAdmin() && !util.IsSudoDaemonServing() {
			if err := util.RunWithElevated(); err != nil {
				log.Warn(err)
				return
			}
		}
		_, err := daemon_client.GetDaemonClient(true)
		if err != nil {
			log.Warn(err)
			return
		}
		client, err := daemon_client.GetDaemonClient(false)
		if err != nil {
			log.Warn(err)
			return
		}
		must(common.Prepare())
		err = client.SendVPNOperateCommand(common.KubeConfig, common.NameSpace, command.Reconnect, workloads, f)
		if err != nil {
			log.Warn(err)
		}
	},
}
