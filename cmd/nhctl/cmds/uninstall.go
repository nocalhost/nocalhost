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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/clientgoutils"
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
		if applicationName == app.DefaultNocalhostApplication {
			log.Error(app.DefaultNocalhostApplicationOperateErr)
			return
		}

		if nameSpace == "" {
			if nameSpace, err = clientgoutils.GetNamespaceFromKubeConfig(kubeConfig); err != nil {
				log.FatalE(err, "Failed to get namespace")
			}
			if nameSpace == "" {
				log.Fatal("Namespace mush be provided")
			}
		}
		if !nocalhost.CheckIfApplicationExist(applicationName, nameSpace) {
			log.Fatalf("Application \"%s\" not found", applicationName)
		}

		log.Info("Uninstalling application...")
		nhApp, err := app.NewApplication(applicationName, nameSpace, kubeConfig, true)
		if err != nil {
			if !force {
				log.FatalE(err, "Failed to get application")
			}
			if err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace); err != nil {
				log.WarnE(err, "Failed to clean up application resource")
			}
			log.Infof("Application \"%s\" is uninstalled anyway.\n", applicationName)
			return
		} else {
			// check if there are services in developing state
			appProfile, err := nhApp.GetProfile()
			if err != nil {
				log.FatalE(err, "")
			}
			for _, profile := range appProfile.SvcProfile {
				if profile.Developing {
					if err = nhApp.StopSyncAndPortForwardProcess(profile.ActualName, true); err != nil {
						log.WarnE(err, "")
					}
				} else if len(profile.DevPortForwardList) > 0 {
					if err = nhApp.StopAllPortForward(profile.ActualName); err != nil {
						log.WarnE(err, "")
					}
				}
			}
		}
		if err = nhApp.Uninstall(force); err != nil {
			if !force {
				log.Fatalf("failed to uninstall application, %v", err)
			}
			if err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace); err != nil {
				log.WarnE(err, "Failed to clean up application resource:")
			}
			return
		}
		log.Infof("Application \"%s\" is uninstalled", applicationName)
	},
}
