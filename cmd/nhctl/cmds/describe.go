/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"fmt"
	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var deploy string

func init() {
	describeCmd.Flags().StringVarP(&deploy, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	describeCmd.Flags().StringVarP(&serviceType, "type", "t", "", "specify service type")
	rootCmd.AddCommand(describeCmd)
}

var describeCmd = &cobra.Command{
	Use:   "describe [NAME]",
	Short: "Describe application info",
	Long:  `Describe application info`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initApp(applicationName)
		if deploy == "" {
			appProfile := nocalhostApp.GetDescription()
			bytes, err := yaml.Marshal(appProfile)
			if err == nil {
				fmt.Print(string(bytes))
			}
		} else {
			checkIfSvcExist(deploy, serviceType)
			appProfile := nocalhostSvc.GetDescription()
			bytes, err := yaml.Marshal(appProfile)
			if err == nil {
				fmt.Print(string(bytes))
			}
		}
	},
}
