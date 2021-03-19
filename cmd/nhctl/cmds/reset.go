/*
Copyright 2021 The Nocalhost Authors.
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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	//resetCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	rootCmd.AddCommand(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset [NAME]",
	Short: "reset application",
	Long:  `reset application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		var err error
		applicationName := args[0]
		initApp(applicationName)

		// Stop BackGroup Process
		for _, profile := range nocalhostApp.AppProfileV2.SvcProfile {
			if profile.Developing {
				err = nocalhostApp.StopSyncAndPortForwardProcess(profile.ActualName, true)
				if err != nil {
					log.WarnE(err, "")
				}
			} else if len(profile.DevPortForwardList) > 0 {
				err = nocalhostApp.StopAllPortForward(profile.ActualName)
				if err != nil {
					log.WarnE(err, "")
				}
			}
		}

		// Remove files
		err = nocalhost.CleanupAppFilesUnderNs(applicationName, nameSpace)
		if err != nil {
			log.WarnE(err, "")
		} else {
			log.Info("Files have been clean up")
		}
		log.Infof("Application %s has been reset.\n", applicationName)
	},
}
