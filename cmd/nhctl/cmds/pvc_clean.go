/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
			exist, err := nocalhostApp.Controller(pvcFlags.Svc, base.Deployment).CheckIfExist()
			if err != nil {
				log.FatalE(err, "failed to check if controller exists")
			} else if !exist {
				log.Fatalf("\"%s\" not found", pvcFlags.Svc)
			}
		}

		mustI(nocalhostApp.CleanUpPVCs(pvcFlags.Svc, true), "Cleaning up pvcs failed")
	},
}
