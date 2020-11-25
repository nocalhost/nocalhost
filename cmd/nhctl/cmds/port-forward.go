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
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
)

//var remotePort string
var portForwardOptions = &app.PortForwardOptions{}

//type PortForwardFlags struct {
//	*EnvSettings
//	LocalPort  int
//	RemotePort int
//	Deployment string
//}
//
//var portForwardFlags = PortForwardFlags{
//	EnvSettings: settings,
//}

func init() {
	portForwardCmd.Flags().IntVarP(&portForwardOptions.LocalPort, "local-port", "l", 0, "local port to forward")
	portForwardCmd.Flags().IntVarP(&portForwardOptions.RemotePort, "remote-port", "r", 0, "remote port to be forwarded")
	portForwardCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	rootCmd.AddCommand(portForwardCmd)
}

var portForwardCmd = &cobra.Command{
	Use:   "port-forward",
	Short: "Forward local port to remote pod'port",
	Long:  `Forward local port to remote pod'port`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		if !nh.CheckIfApplicationExist(applicationName) {
			fmt.Printf("[error] application \"%s\" not found\n", applicationName)
			os.Exit(1)
		}
		nocalhostApp, err = app.NewApplication(applicationName)
		clientgoutils.Must(err)

		if deployment == "" {
			fmt.Println("error: please use -d to specify a kubernetes deployment")
			return
		}

		// todo check deployment if exist

		err = nocalhostApp.SetPortForwardedStatus(true)
		if err != nil {
			fmt.Printf("[error] fail to update \"portForwarded\" status\n")
			os.Exit(1)
		}
		err = nocalhostApp.SshPortForward(deployment, portForwardOptions)
		err2 := nocalhostApp.SetPortForwardedStatus(false)
		if err2 != nil {
			fmt.Printf("[error] fail to update \"portForwarded\" status\n")
			os.Exit(1)
		}
		if err != nil {
			os.Exit(1)
		}

	},
}
