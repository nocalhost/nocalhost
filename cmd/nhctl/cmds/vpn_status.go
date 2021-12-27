/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/log"
	"sigs.k8s.io/yaml"
)

func init() {
	vpnStatusCmd.Flags().StringVar(&kubeConfig, "kubeconfig", clientcmd.RecommendedHomeFile, "kubeconfig")
	vpnStatusCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "namespace")
	vpnStatusCmd.Flags().StringVar(&workloads, "workloads", "", "workloads, like: services/tomcat, deployment/nginx, replicaset/tomcat...")
	vpnStatusCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	vpnCmd.AddCommand(vpnStatusCmd)
}

var vpnStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "status",
	Long:  `status`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := daemon_client.GetDaemonClient(false)
		if err != nil {
			log.Fatal(err)
		}
		result, err := client.SendVPNStatusCommand()
		if err != nil {
			log.Fatal(err)
			return
		}
		marshal, _ := yaml.Marshal(result)
		println(string(marshal))
	},
}
