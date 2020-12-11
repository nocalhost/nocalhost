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
	"fmt"
	"nocalhost/internal/nhctl/app"

	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var force bool

func init() {
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
		if settings.Debug {
			log.SetLevel(logrus.DebugLevel)
		}
		applicationName := args[0]
		if !nh.CheckIfApplicationExist(applicationName) {
			log.Fatalf("application \"%s\" not found\n", applicationName)
		}

		fmt.Println("uninstalling application...")
		nhApp, err := nh.GetApplication(applicationName)
		if err != nil {
			if !force {
				log.Fatalf("failed to get application, %v", err)
			} else {
				err = nh.CleanupAppFiles(applicationName)
				if err != nil {
					log.Warnf("fail to clean up application resource: %s", err.Error())
				}
				fmt.Printf("application \"%s\" is uninstalled anyway.\n", applicationName)
				return
			}
		} else {
			// check if there are services in developing state
			for _, profile := range nhApp.AppProfile.SvcProfile {
				if profile.Developing {
					log.Debugf("end %s dev model", profile.ActualName)
					err = nhApp.EndDevelopMode(profile.ActualName, &app.FileSyncOptions{})
					if err != nil {
						log.Warnf("fail to end %s dev model: %s", err.Error())
					}
				}
			}
		}
		err = nhApp.Uninstall(force)
		if err != nil {
			log.Fatalf("failed to uninstall application, %v", err)
		}
		fmt.Printf("application \"%s\" is uninstalled\n", applicationName)
	},
}
