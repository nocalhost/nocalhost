/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/common/base"
)

var deploy string

func init() {
	describeCmd.Flags().StringVarP(&deploy, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	describeCmd.Flags().StringVarP(&common.ServiceType, "type", "t", "deployment", "specify service type")
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
		nocalhostApp, err := common.InitApp(applicationName)
		must(err)
		if deploy == "" {
			appProfile := nocalhostApp.GetDescription()
			for _, svcProfileV2 := range appProfile.SvcProfile {
				svcProfileV2.DevModeType = nocalhostApp.GetAppMeta().GetCurrentDevModeTypeOfWorkload(svcProfileV2.Name, base.SvcType(svcProfileV2.Type), appProfile.Identifier)
			}
			bytes, err := yaml.Marshal(appProfile)
			if err == nil {
				fmt.Print(string(bytes))
			}
		} else {
			nocalhostSvc, err := nocalhostApp.InitAndCheckIfSvcExist(deploy, common.ServiceType)
			must(err)
			svcProfile := nocalhostSvc.GetDescription()
			bytes, err := yaml.Marshal(svcProfile)
			if err == nil {
				fmt.Print(string(bytes))
			}
		}
	},
}
