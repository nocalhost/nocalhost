/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	devResetCmd.Flags().StringVarP(&deployment, "deployment", "d", "",
		"k8s deployment which your developing service exists")
	devResetCmd.Flags().StringVarP(&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet")
	debugCmd.AddCommand(devResetCmd)
}

var devResetCmd = &cobra.Command{
	Use:   "reset [NAME]",
	Short: "reset service",
	Long:  `reset service`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		_ = nocalhostSvc.DevEnd(true)

		log.Infof("Service %s has been reset.\n", deployment)
	},
}
