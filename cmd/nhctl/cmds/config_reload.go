/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	configReloadCmd.Flags().StringVarP(
		&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	configReloadCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	configCmd.AddCommand(configReloadCmd)
}

var configReloadCmd = &cobra.Command{
	Use:   "reload [Name]",
	Short: "reload application/service config",
	Long:  "reload application/service config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		nocalhostApp, err := common.InitApp(commonFlags.AppName)
		must(err)

		if commonFlags.SvcName == "" {
			if err := nocalhostApp.ReloadCfg(true, false); err != nil {
				log.Fatal(errors.Wrap(err, ""))
			}
		} else {
			nocalhostSvc, err := nocalhostApp.InitAndCheckIfSvcExist(commonFlags.SvcName, common.ServiceType)
			must(err)
			if err := nocalhostApp.ReloadSvcCfg(commonFlags.SvcName, nocalhostSvc.Type, true, false); err != nil {
				log.Fatal(errors.Wrap(err, ""))
			}
		}
	},
}
