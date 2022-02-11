/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/pkg/nhctl/clientgoutils"
	"path/filepath"

	"nocalhost/pkg/nhctl/log"
)

func init() {
	pvcCleanCmd.Flags().StringVar(&pvcFlags.App, "app", "", "Clean up PVCs of specified application")
	pvcCleanCmd.Flags().StringVar(&pvcFlags.Svc, "controller", "", "Clean up PVCs of specified service")
	pvcCleanCmd.Flags().StringVar(&pvcFlags.Name, "name", "", "Clean up specified PVC")
	pvcCleanCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	pvcCmd.AddCommand(pvcCleanCmd)
}

var pvcCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up PersistVolumeClaims",
	Long:  `Clean up PersistVolumeClaims`,
	Run: func(cmd *cobra.Command, args []string) {

		// Clean up specified pvc
		if pvcFlags.Name != "" {
			if abs, err := filepath.Abs(common.KubeConfig); err == nil {
				common.KubeConfig = abs
			}
			cli, err := clientgoutils.NewClientGoUtils(common.KubeConfig, common.NameSpace)
			must(err)
			mustI(cli.DeletePVC(pvcFlags.Name), "Failed to clean up pvc: "+pvcFlags.Name)
			log.Infof("Persistent volume %s has been cleaned up", pvcFlags.Name)
			return
		}

		if pvcFlags.App == "" {
			// Clean up all pvcs in namespace
			cli, err := clientgoutils.NewClientGoUtils(common.KubeConfig, common.NameSpace)
			must(err)
			pvcList, err := cli.ListPvcs()
			must(err)
			if len(pvcList) == 0 {
				log.Info("No pvc found")
			}
			for _, pvc := range pvcList {
				must(cli.DeletePVC(pvc.Name))
				log.Infof("Persistent volume %s has been cleaned up", pvc.Name)
			}
			return
		}

		var (
			pvcs []v1.PersistentVolumeClaim
			err  error
		)

		var nocalhostApp *app.Application
		var nocalhostSvc *controller.Controller
		// Clean up PVCs of specified service
		if pvcFlags.Svc != "" {
			nocalhostApp, nocalhostSvc, err = common.InitAppAndCheckIfSvcExist(pvcFlags.App, pvcFlags.Svc, common.ServiceType)
			must(err)
			pvcs, err = nocalhostSvc.GetPVCsBySvc()
		} else {
			// Clean up all pvcs in application
			nocalhostApp, err = common.InitApp(pvcFlags.App)
			must(err)
			pvcs, err = nocalhostApp.GetAllPVCs()
		}

		must(err)

		if len(pvcs) == 0 {
			log.Info("No Persistent volume needs to be cleaned up")
		}

		// todo check if pvc still is used by some pods
		for _, pvc := range pvcs {
			err = nocalhostApp.GetClient().DeletePVC(pvc.Name)
			if err != nil {
				log.WarnE(err, fmt.Sprintf("error occurs while deleting persistent volume %s", pvc.Name))
			} else {
				log.Infof("Persistent volume %s has been cleaned up", pvc.Name)
			}
		}
	},
}
