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
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"strings"
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
		InitAppAndSvc(applicationName, deployment)

		if !nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("\"%s\" is not in developing status", deployment)
		}

		fmt.Println("exiting dev model...")
		// end file sync
		log.Debug("ending file sync...")
		EndFileSync()
		err = nocalhostApp.SetSyncingStatus(deployment, false)
		if err != nil {
			log.Fatal("fail to update \"syncing\" status")
		}

		log.Debug("stopping port-forward...")
		err = nocalhostApp.StopAllPortForward(deployment)
		if err != nil {
			fmt.Printf("[warning] fail to stop port forward, %v\n", err)
		}
		err = nocalhostApp.SetPortForwardedStatus(deployment, false)
		if err != nil {
			log.Fatal("fail to update \"portForwarded\" status")
		}

		log.Debug("roll back deployment...")
		err = nocalhostApp.RollBack(context.TODO(), deployment)
		if err != nil {
			log.Fatal("fail to rollback")
		}
		err = nocalhostApp.SetDevelopingStatus(deployment, false)
		if err != nil {
			log.Fatal("fail to update \"developing\" status")
		}
		fmt.Printf("%s have exited develop model\n", deployment)
	},
}

func EndFileSync() {
	output, _ := tools.ExecCommand(nil, false, "mutagen", "sync", "list")
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Name") {
			strs := strings.Split(line, ":")
			if len(strs) >= 2 {
				sessionName := strings.TrimLeft(strs[1], " ")
				fmt.Printf("terminate sync session :%s \n", sessionName)
				_, err := tools.ExecCommand(nil, true, "mutagen", "sync", "terminate", sessionName)
				if err != nil {
					printlnErr("failed to terminate sync session", err)
				} else {
					// todo confirm session's status
					fmt.Println("sync session has been terminated.")
				}
			}
		}
	}
}
