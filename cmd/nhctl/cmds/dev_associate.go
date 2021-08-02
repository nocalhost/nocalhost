/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/log"
)

var workDir string
var deAssociate bool

func init() {
	devAssociateCmd.Flags().StringVarP(
		&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	devAssociateCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	devAssociateCmd.Flags().StringVarP(&workDir, "associate", "s", "", "dev mode work directory")
	devAssociateCmd.Flags().BoolVar(&deAssociate, "de-associate", false, "de associate(for test)")
	debugCmd.AddCommand(devAssociateCmd)
}

var devAssociateCmd = &cobra.Command{
	Use:   "associate [Name]",
	Short: "associate service dev dir",
	Long:  "associate service dev dir",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		initApp(commonFlags.AppName)

		checkIfSvcExist(commonFlags.SvcName, serviceType)

		if deAssociate {
			must(nocalhostSvc.Associate(""))
		} else {
			if workDir == "" {
				log.Fatal("associate must specify")
			}
			must(nocalhostSvc.Associate(workDir))
		}

		must(nocalhostApp.ReloadSvcCfg(nocalhostSvc.Name, nocalhostSvc.Type, false, false))
	},
}
