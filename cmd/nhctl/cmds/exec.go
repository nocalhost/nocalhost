/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ExecFlags struct {
	CommonFlags
	Commands  []string
	Container string
}

var execFlags = ExecFlags{}

func init() {
	execCmd.Flags().StringArrayVarP(&execFlags.Commands, "command", "c", nil,
		"command to execute in container")
	execCmd.Flags().StringVarP(&execFlags.Container, "container", "", "", "container name")
	execCmd.Flags().StringVarP(&execFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists")
	execCmd.Flags().StringVarP(&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet")
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
		initAppAndCheckIfSvcExist(execFlags.AppName, execFlags.SvcName, serviceType)
		must(nocalhostApp.Exec(execFlags.SvcName, execFlags.Container, execFlags.Commands))
	},
}
