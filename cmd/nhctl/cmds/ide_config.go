/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/pkg/nhctl/log"
)

var configAction string

func init() {
	ideConfigCmd.Flags().StringVarP(&common.WorkloadName, "deployment", "d", "",
		"k8s deployment your developing service exists")
	ideConfigCmd.Flags().StringVarP(&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
	ideConfigCmd.Flags().StringVarP(&configAction, "action", "a", "",
		"action applied in nocalhost config, such as check")
	ideCmd.AddCommand(ideConfigCmd)
}

var ideConfigCmd = &cobra.Command{
	Use:   "config [NAME]",
	Short: "crud for nocalhost config",
	Long:  `crud for nocalhost config`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if configAction == "check" {
			applicationName := args[0]
			_, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)
			must(err)

			c := nocalhostSvc.Config()
			for _, containerConfig := range c.ContainerConfigs {
				if containerConfig.Dev != nil && containerConfig.Dev.Image != "" {
					fmt.Print("true")
					return
				}
			}
			fmt.Print("false")
		} else {
			log.Fatalf("Unsupported action %s", configAction)
		}

	},
}
