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
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing"
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

		fmt.Println("Ending dev mode...")
		// end file sync
		fmt.Println("ending file sync...")
		// get dev-start stage record free pod so it do not need get free port agian
		var devStartOptions = &app.DevStartOptions{}
		fileSyncOps, err = nocalhostApp.GetSyncthingPort(deployment, fileSyncOps)
		if err != nil {
			fmt.Println("[error] fail to get syncthing port, you can start sync command first")
		}

		newSyncthing, err := syncthing.New(nocalhostApp, deployment, devStartOptions, fileSyncOps)
		if err != nil {
			log.Fatalf("[error] fail to new syncthing")
		}
		// read and empty pid file
		portForwardPid, portForwardFilePath, err := nocalhostApp.GetBackgroundSyncPortForwardPid(deployment, false)
		if err != nil {
			fmt.Println("[info] fail to get background port-forward pid file, ignore")
		}
		if portForwardPid != 0 {
			err = newSyncthing.Stop(portForwardPid, portForwardFilePath, "port-forward", true)
		}
		if err != nil {
			fmt.Printf("[info] fail stop port-forward progress pid %d, please run `kill -9 %d` by manual\n", portForwardPid, portForwardPid)
		}

		// read and empty pid file
		syngthingPid, syncThingPath, err := nocalhostApp.GetBackgroundSyncThingPid(deployment, false)
		if err != nil {
			fmt.Println("[info] fail to get background port-forward pid file, ignore")
		}
		if syngthingPid != 0 {
			err = newSyncthing.Stop(syngthingPid, syncThingPath, "syncthing", true)
		}
		if err != nil {
			fmt.Printf("[info] fail stop syncthing progress pid %d, please run `kill -9 %d` by manual\n", portForwardPid, portForwardPid)
		}

		if err == nil { // none of them has error
			fmt.Printf("[info] exit background port-forward: %d syncthing: %d \n", portForwardPid, syngthingPid)
		}

		// end dev port background port forward
		// read and empty pid file
		onlyPortForwardPid, onlyPortForwardFilePath, err := nocalhostApp.GetBackgroundOnlyPortForwardPid(deployment, false)
		if err != nil {
			fmt.Println("[info] none of dev port-forward pid file, ignore")
		}
		if onlyPortForwardPid != 0 {
			err = newSyncthing.Stop(onlyPortForwardPid, onlyPortForwardFilePath, "port-forward", true)
		}
		if err != nil {
			fmt.Printf("[info] fail to ending dev port-forward pid %d, please run `kill -9 %d` by manual\n", onlyPortForwardPid, onlyPortForwardPid)
		}

		if err == nil {
			fmt.Printf("[info] exit dev port-forward: %d \n", onlyPortForwardPid)
		}

		// set profile status
		// set port-forward port and ignore result
		_ = nocalhostApp.SetSyncthingPort(deployment, 0, 0, 0, 0)
		// roll back deployment
		log.Debug("roll back deployment...")
		err = nocalhostApp.RollBack(context.TODO(), deployment)
		if err != nil {
			fmt.Printf("fail to rollback")
		}
		err = nocalhostApp.SetDevEndProfileStatus(deployment)
		if err != nil {
			log.Fatal("fail to update \"developing\" status")
		}
		fmt.Printf("%s dev mode ended\n", deployment)
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
