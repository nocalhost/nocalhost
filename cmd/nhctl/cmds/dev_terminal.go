/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"nocalhost/cmd/nhctl/cmds/common"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
)

var pod string
var shell string

func init() {
	devTerminalCmd.Flags().StringVarP(
		&common.WorkloadName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	devTerminalCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	devTerminalCmd.Flags().StringVarP(&container, "container", "c", "", "container to enter")
	devTerminalCmd.Flags().StringVar(&pod, "pod", "", "pod to enter")
	devTerminalCmd.Flags().StringVarP(
		&shell, "shell", "", "",
		"shell cmd while enter dev container",
	)
	debugCmd.AddCommand(devTerminalCmd)
}

var devTerminalCmd = &cobra.Command{
	Use:   "terminal [NAME]",
	Short: "Enter dev container's terminal",
	Long:  `Enter dev container's terminal`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		_, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)
		must(err)

		if pod == "" {
			podList, err := nocalhostSvc.GetPodList()
			must(err)
			var runningPod = make([]v1.Pod, 0, 1)

		Exit:
			for _, item := range podList {
				if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
					runningPod = append(runningPod, item)
				}

				for _, c := range item.Spec.Containers {
					if c.Name == _const.NocalhostDefaultDevSidecarName {
						pod = item.Name
						break Exit
					}
				}
			}
			if pod == "" && len(runningPod) != 1 {
				log.Fatalf("Pod num is %d (not 1), please specify one", len(runningPod))
			}
			pod = runningPod[0].Name
		}
		must(nocalhostSvc.EnterPodTerminal(pod, container, shell, ""))
	},
}
