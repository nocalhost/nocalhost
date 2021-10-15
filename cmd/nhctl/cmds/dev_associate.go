/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/pkg/nhctl/log"
	"os"
)

var workDir string
var workDirDeprecated string
var deAssociate bool
var info bool
var migrate bool

func init() {
	devAssociateCmd.Flags().StringVarP(
		&workDir, "local-sync", "s", "",
		"the local directory synchronized to the remote container under dev mode",
	)
	devAssociateCmd.Flags().StringVar(
		&workDirDeprecated, "associate", "",
		"the local directory synchronized to the remote container under dev mode(deprecated)",
	)
	devAssociateCmd.Flags().StringVarP(
		&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	devAssociateCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	devAssociateCmd.Flags().StringVarP(
		&container, "container", "c", "",
		"container to develop",
	)
	devAssociateCmd.Flags().BoolVar(
		&deAssociate, "de-associate", false,
		"[exclusive with info flag] de associate a svc from associated work dir",
	)
	devAssociateCmd.Flags().BoolVar(
		&migrate, "migrate", false,
		"associate the local directory synchronized but with low priority",
	)
	devAssociateCmd.Flags().BoolVar(
		&info, "info", false,
		"get associated path from svc ",
	)
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

		must(Prepare())

		svcPack := dev_dir.NewSvcPack(
			nameSpace,
			commonFlags.AppName,
			base.SvcTypeOf(serviceType),
			commonFlags.SvcName,
			container,
		)

		if info {
			fmt.Printf(svcPack.GetAssociatePath().ToString())
			return
		} else if deAssociate {
			svcPack.UnAssociatePath()
			return
		}
		if workDirDeprecated != "" {
			workDir = workDirDeprecated
		}

		if workDir == "" {
			log.Fatal("--local-sync must specify")
		}

		initApp(commonFlags.AppName)
		checkIfSvcExist(commonFlags.SvcName, serviceType)

		if !dev_dir.DevPath(workDir).AlreadyAssociate(svcPack) && ((nocalhostSvc.IsInReplaceDevMode() && nocalhostSvc.IsProcessor()) || nocalhostSvc.IsInDuplicateDevMode()) {
			log.PWarn("Current svc is already in DevMode, so can not switch associate dir, please exit the DevMode and try again.")
			os.Exit(1)
		}

		must(dev_dir.DevPath(workDir).Associate(svcPack, kubeConfig, !migrate))
		must(nocalhostApp.ReloadSvcCfg(nocalhostSvc.Name, nocalhostSvc.Type, false, false))
	},
}
