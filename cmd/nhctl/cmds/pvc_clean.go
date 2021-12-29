/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/pkg/nhctl/clientgoutils"
	"path/filepath"

	"nocalhost/pkg/nhctl/log"
)

func init() {
	pvcCleanCmd.Flags().StringVar(&pvcFlags.App, "app", "", "Clean up PVCs of specified application")
	pvcCleanCmd.Flags().StringVar(&pvcFlags.Svc, "controller", "", "Clean up PVCs of specified service")
	pvcCleanCmd.Flags().StringVar(&pvcFlags.Name, "name", "", "Clean up specified PVC")
	pvcCmd.AddCommand(pvcCleanCmd)
}

var pvcCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up PersistVolumeClaims",
	Long:  `Clean up PersistVolumeClaims`,
	Run: func(cmd *cobra.Command, args []string) {

		// Clean up specified pvc
		if pvcFlags.Name != "" {
			if abs, err := filepath.Abs(kubeConfig); err == nil {
				kubeConfig = abs
			}
			cli, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
			must(err)
			mustI(cli.DeletePVC(pvcFlags.Name), "Failed to clean up pvc: "+pvcFlags.Name)
			log.Infof("Persistent volume %s has been cleaned up", pvcFlags.Name)
			return
		}

		if pvcFlags.App == "" {
			// Clean up all pvcs in namespace
			cli, err := clientgoutils.NewClientGoUtils(kubeConfig, nameSpace)
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

		// Clean up all pvcs in application
		initApp(pvcFlags.App)

		// Clean up PVCs of specified service
		if pvcFlags.Svc != "" {
			c, err := nocalhostApp.Controller(pvcFlags.Svc, base.Deployment)
			if err != nil {
				log.FatalE(err, "")
			}
			if err = c.CheckIfExist(); err != nil {
				log.FatalE(err, "")
			}
		}

		mustI(nocalhostApp.CleanUpPVCs(pvcFlags.Svc, true), "Cleaning up pvcs failed")
	},
}
