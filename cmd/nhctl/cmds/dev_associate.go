/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
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
var nid string

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
		&common.ServiceType, "controller-type", "t", "deployment",
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

		var err error = nil
		var nid = ""
		if err = common.Prepare(); err == nil {
			if err = common.InitApp(commonFlags.AppName); err == nil {
				if err := common.CheckIfSvcExist(commonFlags.SvcName, common.ServiceType); err == nil {
					nid = common.NocalhostSvc.AppMeta.NamespaceId
				}
			}
		}

		svcPack := dev_dir.NewSvcPack(
			common.NameSpace,
			nid,
			commonFlags.AppName,
			base.SvcType(common.ServiceType),
			commonFlags.SvcName,
			container,
		)

		if info {
			fmt.Print(svcPack.GetAssociatePath().ToString())
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

		must(err)

		if (common.NocalhostSvc.IsInReplaceDevMode() && common.NocalhostSvc.IsProcessor()) || common.NocalhostSvc.IsInDuplicateDevMode() {
			if !dev_dir.DevPath(workDir).AlreadyAssociate(svcPack) {
				log.PWarn("Current svc is already in DevMode, so can not switch associate dir, please exit the DevMode and try again.")
				os.Exit(1)
			} else {
				if profile, err := common.NocalhostSvc.GetProfile(); err != nil {
					log.PWarn("Fail to get profile of current svc, please exit the DevMode and try again.")
					os.Exit(1)
				} else {
					svcPack = dev_dir.NewSvcPack(
						common.NameSpace,
						nid,
						commonFlags.AppName,
						base.SvcType(common.ServiceType),
						commonFlags.SvcName,
						profile.OriginDevContainer,
					)
				}
			}
		}

		must(dev_dir.DevPath(workDir).Associate(svcPack, common.KubeConfig, !migrate))

		must(common.NocalhostApp.ReloadSvcCfg(common.NocalhostSvc.Name, common.NocalhostSvc.Type, false, false))
	},
}
