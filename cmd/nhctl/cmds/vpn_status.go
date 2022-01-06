/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
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
		var n name
		if client, err := daemon_client.GetDaemonClient(false); err == nil {
			if command, err := client.SendVPNStatusCommand(); err == nil {
				if marshal, err := json.Marshal(command); err == nil {
					var result cluster
					if err = json.Unmarshal(marshal, &result); err == nil {
						n.Actual = result
					}
				}
			}
		}
		if sudoclient, err := daemon_client.GetDaemonClient(true); err == nil {
			if command, err := sudoclient.SendSudoVPNStatusCommand(); err == nil {
				if marshal, err := json.Marshal(command); err == nil {
					var result pkg.ConnectOptions
					if err = json.Unmarshal(marshal, &result); err == nil {
						n.Expected = cluster{
							Namespace:  result.Namespace,
							Kubeconfig: string(result.KubeconfigBytes),
						}
					}
				}
			}
		}
		n.isEquals()
		marshal, _ := yaml.Marshal(n)
		println(string(marshal))
	},
}

type name struct {
	Expected cluster
	Actual   cluster
	Equal    bool
}

func (n *name) isEquals() {
	n.Equal = util.GenerateKey([]byte(n.Actual.Kubeconfig), n.Actual.Namespace) ==
		util.GenerateKey([]byte(n.Expected.Kubeconfig), n.Expected.Namespace)
}

type cluster struct {
	Namespace  string
	Kubeconfig string
}
