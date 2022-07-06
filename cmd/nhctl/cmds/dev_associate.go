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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/profile"
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
	devAssociateCmd.Flags().StringVarP(
		&nid, "nid", "", "",
		"namespace id",
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

		// init this two field logic:
		// if nid is not empty, delay init when use it
		// if nid is empty, init it immediately
		var nocalhostApp *app.Application
		var nocalhostSvc *controller.Controller

		var f = func() {
			if err = common.Prepare(); err == nil {
				if nocalhostApp, err = common.InitApp(commonFlags.AppName); err == nil {
					if nocalhostSvc, err = nocalhostApp.InitAndCheckIfSvcExist(commonFlags.SvcName, common.ServiceType); err == nil {
						nid = nocalhostSvc.AppMeta.NamespaceId
					}
				}
			}
		}
		if len(nid) == 0 {
			f()
		}

		svcPack := dev_dir.NewSvcPack(
			common.NameSpace,
			nid,
			commonFlags.AppName,
			base.SvcType(common.ServiceType),
			commonFlags.SvcName,
			container,
		)

		// notify daemon to invalid cache before reture
		defer func() {
			if client, err := daemon_client.GetDaemonClient(false); err == nil {
				_ = client.SendFlushDirMappingCacheCommand(common.NameSpace, nid, commonFlags.AppName)
			}
		}()

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

		// needs to init if not have init
		if nocalhostApp == nil || nocalhostSvc == nil {
			f()
		}

		if (nocalhostSvc.IsInReplaceDevMode() && nocalhostSvc.IsProcessor()) || nocalhostSvc.IsInDuplicateDevMode() {
			if !dev_dir.DevPath(workDir).AlreadyAssociate(svcPack) {
				log.PWarn("Current svc is already in DevMode, so can not switch associate dir, please exit the DevMode and try again.")
				os.Exit(1)
			} else {
				if profile, err := nocalhostSvc.GetProfile(); err != nil {
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
		must(
			nocalhostSvc.UpdateSvcProfile(
				func(v2 *profile.SvcProfileV2) error {
					return nil
				},
			),
		)

		must(nocalhostApp.ReloadSvcCfg(nocalhostSvc.Name, nocalhostSvc.Type, false, false))
	},
}
