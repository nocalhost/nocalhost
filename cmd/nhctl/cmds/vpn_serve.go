/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
)

var config pkg.Route

func init() {
	ServerCmd.Flags().StringArrayVarP(&config.ServeNodes, "node", "L", []string{}, "server node")
	ServerCmd.Flags().StringVarP(&config.ChainNode, "chain", "F", "", "forward chain node")
	ServerCmd.Flags().BoolVar(&util.Debug, "debug", false, "true/false")
	vpnCmd.AddCommand(ServerCmd)
}

var ServerCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve",
	Long:  `serve`,
	PreRun: func(*cobra.Command, []string) {
		util.InitLogger(util.Debug)
	},
	Run: func(cmd *cobra.Command, args []string) {
		c, err := pkg.Start(context.TODO(), config)
		if err != nil {
			log.Fatal(err)
		}
		if err := <-c; err != nil {
			log.Fatal(err)
		}
	},
}
