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
	"nocalhost/internal/nhctl/appmeta"
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
			err = cli.DeletePVC(pvcFlags.Name)
			mustI(err, "Failed to clean up pvc: "+pvcFlags.Name)
			log.Infof("%s cleaned up", pvcFlags.Name)
			return
		}

		if pvcFlags.App == "" {
			log.Fatal("--app mush be specified")
		}

		initApp(pvcFlags.App)

		// Clean up PVCs of specified service
		if pvcFlags.Svc != "" {
			exist, err := nocalhostApp.Controller(pvcFlags.Svc, appmeta.Deployment).CheckIfExist()
			if err != nil {
				log.FatalE(err, "failed to check if controller exists")
			} else if !exist {
				log.Fatalf("\"%s\" not found", pvcFlags.Svc)
			}
		}

		mustI(nocalhostApp.CleanUpPVCs(pvcFlags.Svc, true), "Cleaning up pvcs failed")
	},
}
