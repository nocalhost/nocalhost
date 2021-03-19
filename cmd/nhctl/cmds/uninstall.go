/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
			nameSpace, err = clientgoutils.GetNamespaceFromKubeConfig(kubeConfig)
			if err != nil {
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
		nhApp, err := app.NewApplication(applicationName, nameSpace, true)
		if err != nil {
			if !force {
				log.FatalE(err, "Failed to get application")
			} else {
				err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace)
				if err != nil {
					log.WarnE(err, "Failed to clean up application resource")
				}
				log.Infof("Application \"%s\" is uninstalled anyway.\n", applicationName)
				return
			}
		} else {
			// check if there are services in developing state
			for _, profile := range nhApp.AppProfileV2.SvcProfile {
				if profile.Developing {
					err = nhApp.StopSyncAndPortForwardProcess(profile.ActualName, true)
					if err != nil {
						log.WarnE(err, "")
					}
				} else if len(profile.DevPortForwardList) > 0 {
					err = nhApp.StopAllPortForward(profile.ActualName)
					if err != nil {
						log.WarnE(err, "")
					}
				}
			}
		}
		err = nhApp.Uninstall(force)
		if err != nil {
			if force {
				err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace)
				if err != nil {
					log.Warnf("Failed to clean up application resource: %s", err.Error())
				}
				return
			}
			log.Fatalf("failed to uninstall application, %v", err)
		}
		log.Infof("Application \"%s\" is uninstalled", applicationName)
	},
}
