/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"nocalhost/pkg/nhctl/log"
)

//var container string

func init() {
	devContainersCmd.Flags().StringVarP(
		&deployment, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	devContainersCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	debugCmd.AddCommand(devContainersCmd)
}

var devContainersCmd = &cobra.Command{
	Use:   "containers [NAME]",
	Short: "Get containers of workload",
	Long:  `Get containers of workload`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		if pod == "" {
			podList, err := nocalhostSvc.BuildPodController().GetPodList()
			must(err)
			var runningPod = make([]v1.Pod, 0, 1)
			for _, item := range podList {
				if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
					runningPod = append(runningPod, item)
				}
			}
			if len(runningPod) != 1 {
				log.Fatal("Pod num is not 1, please specify one")
			}
			pod = runningPod[0].Name
		}
		must(nocalhostSvc.EnterPodTerminal(pod, container, shell))
	},
}
