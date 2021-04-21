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
	"nocalhost/internal/nhctl/daemon_server"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	daemonStartCmd.Flags().BoolVar(&isSudoUser, "sudo", false, "Is run as sudo")
	daemonStartCmd.Flags().BoolVarP(&runInBackground, "daemon", "d", false, "Is run as daemon(background)")
	daemonCmd.AddCommand(daemonStartCmd)
}

// This command is run by daemon client as a background progress usually
var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start nhctl daemon",
	Long:  `Start nhctl daemon`,
	Run: func(cmd *cobra.Command, args []string) {
		log.AddField("APP", "daemon-server")
		if runInBackground {
			must(daemon_client.StartDaemonServer(isSudoUser))
			return
		}
		must(daemon_server.StartDaemon(isSudoUser, Version, GitCommit))
	},
}
