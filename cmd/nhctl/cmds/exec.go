/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
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
	"regexp"
)

type ExecFlags struct {
	CommonFlags
	Commands  []string
	Container string
}

var execFlags = ExecFlags{}

func init() {
	execCmd.Flags().StringArrayVarP(
		&execFlags.Commands, "command", "c", nil,
		"command to execute in container",
	)
	execCmd.Flags().StringVarP(&execFlags.Container, "container", "", "", "container name")
	execCmd.Flags().StringVarP(
		&execFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	execCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	rootCmd.AddCommand(execCmd)
}

var execCmd = &cobra.Command{
	Use:   "exec [NAME]",
	Short: "Execute a command in container",
	Long:  `Execute a command in container`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		execFlags.AppName = args[0]
		// replace $(XXX) --> ${XXX}, support environment variable
		compile, _ := regexp.Compile(`\$\((.*?)\)`)
		for i := 0; i < len(execFlags.Commands); i++ {
			execFlags.Commands[i] = compile.ReplaceAllString(execFlags.Commands[i], "${$1}")
		}
		nocalhostApp, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(execFlags.AppName, execFlags.SvcName, common.ServiceType)
		must(err)

		podList, err := nocalhostSvc.GetPodList()
		if err != nil {
			log.Fatal(err)
		}
		runningPod := make([]v1.Pod, 0, 1)

		var ableToExec *v1.Pod = nil

	Exit:
		for _, item := range podList {
			if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
				runningPod = append(runningPod, item)
			}

			for _, c := range item.Spec.Containers {
				if c.Name == _const.NocalhostDefaultDevSidecarName {
					ableToExec = &item
					break Exit
				}
			}
		}
		if ableToExec == nil && len(runningPod) != 1 {
			log.Fatalf("pod number: %d, is not 1, please make sure pod number is 1", len(runningPod))
		}
		ableToExec = &runningPod[0]
		must(nocalhostApp.Exec(*ableToExec, execFlags.Container, execFlags.Commands))
	},
}
