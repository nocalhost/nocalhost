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
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	daemonRestartCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonCmd.AddCommand(daemonRestartCmd)
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart nhctl daemon",
	Long:  `Restart nhctl daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		isRunning := daemon_client.CheckIfDaemonServerRunning(isSudoUser)
		if isRunning {
			client, err := daemon_client.NewDaemonClient(isSudoUser)
			must(err)
			must(client.SendRestartDaemonServerCommand())
			log.Info("RestartDaemonServerCommand has been sent")
		} else {
			log.Warnf("Daemon Server(sudo:%t) is not running", isSudoUser)
			must(daemon_client.StartDaemonServer(isSudoUser))
		}
	},
}
