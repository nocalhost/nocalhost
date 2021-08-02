/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var force bool

func init() {
	uninstallCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	uninstallCmd.Flags().BoolVar(&force, "force", false, "force to uninstall anyway")
	rootCmd.AddCommand(uninstallCmd)
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [NAME]",
	Short: "Uninstall application",
	Long:  `Uninstall application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		if applicationName == _const.DefaultNocalhostApplication {
			log.Error(_const.DefaultNocalhostApplicationOperateErr)
			return
		}

		must(Prepare())

		appMeta, err := nocalhost.GetApplicationMeta(applicationName, nameSpace, kubeConfig)
		must(err)

		if appMeta.IsNotInstall() {
			log.Fatalf(appMeta.NotInstallTips())
			return
		}

		log.Info("Uninstalling application...")

		//goland:noinspection ALL
		mustI(appMeta.Uninstall(true), "Error while uninstall application")

		log.Infof("Application \"%s\" is uninstalled", applicationName)
	},
}
