/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	devPodCmd.Flags().StringVarP(
		&deployment, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	devPodCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	debugCmd.AddCommand(devPodCmd)
}

var devPodCmd = &cobra.Command{
	Use:   "pod [NAME]",
	Short: "Get pod of workload",
	Long:  `Get pod of workload`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		podList, err := nocalhostSvc.BuildPodController().GetPodList()
		if err != nil || len(podList) != 1 {
			return
		}
		fmt.Println(podList[0].Name)
	},
}
