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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"nocalhost/pkg/nhctl/log"
)

func init() {
	devEndCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	debugCmd.AddCommand(devEndCmd)
}

var devEndCmd = &cobra.Command{
	Use:   "end [NAME]",
	Short: "end dev model",
	Long:  `end dev model`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, nil)

		if !nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("Service %s is not in DevMode", deployment)
		}

		log.Info("Terminating file sync process...")
		err = nocalhostApp.StopSyncAndPortForwardProcess(deployment)
		if err != nil {
			log.WarnE(err, "Error occurs when stopping sync process")
		}
		err = nocalhostApp.EndDevelopMode(deployment)
		if err != nil {
			log.FatalE(err, fmt.Sprintf("Failed to end %s", deployment))
		}
		log.Infof("Service %s's DevMode has been ended", deployment)
	},
}
