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
	"os"
	"os/exec"
	"strconv"
	"strings"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var portForwardOptions = &app.PortForwardOptions{}

func init() {
	portForwardCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	portForwardCmd.Flags().StringSliceVarP(&portForwardOptions.DevPort, "dev-port", "p", []string{}, "port-forward between pod and local, such 8080:8080 or :8080(random localPort)")
	portForwardCmd.Flags().BoolVarP(&portForwardOptions.RunAsDaemon, "daemon", "m", true, "if port-forward run as daemon")
	rootCmd.AddCommand(portForwardCmd)
}

var portForwardCmd = &cobra.Command{
	Use:   "port-forward [NAME]",
	Short: "Forward local port to remote pod's port",
	Long:  `Forward local port to remote pod's port`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// var err error
		applicationName := args[0]
		InitAppAndCheckIfSvcExist(applicationName, deployment)

		if !nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("\"%s\" is not in DevMode", deployment)
		}

		if nocalhostApp.CheckIfSvcIsPortForwarded(deployment) {
			log.Fatalf("\"%s\" has in port forwarding", deployment)
		}

		// look for nhctl
		NhctlAbsdir, err := exec.LookPath(nocalhostApp.GetMyBinName())
		if err != nil {
			log.Fatal("installing fortune is in your future")
		}
		// overwrite Args[0] as ABS directory of bin directory
		os.Args[0] = NhctlAbsdir

		if portForwardOptions.RunAsDaemon {
			_, err := daemon.Background(nocalhostApp.GetPortForwardLogFile(deployment), nocalhostApp.GetApplicationBackGroundOnlyPortForwardPidFile(deployment), true)
			if err != nil {
				log.Fatalf("failed to run port-forward background, please try again")
			}
		}

		// find deployment pods
		podName := ""
		podNameSpace := ""
		podsList, err := nocalhostApp.GetPodsFromDeployment(deployment)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if podsList != nil {
			// get first pod
			podName = podsList.Items[0].Name
			podNameSpace = podsList.Items[0].Namespace
		}

		// run in child process
		if len(portForwardOptions.DevPort) == 0 {
			// if not specify -p then use default
			portForwardOptions.DevPort = nocalhostApp.GetDefaultDevPort(deployment)
			fmt.Printf("get default devPort: %s \n", portForwardOptions.DevPort)
		}
		var localPorts []int
		var remotePorts []int
		for _, port := range portForwardOptions.DevPort {
			// 8080:8080, :8080
			s := strings.Split(port, ":")
			fmt.Printf("split: %s \n", s)
			if len(s) < 2 {
				// ignore wrong format
				fmt.Printf("wrong format of dev port:%s , skipped. \n", port)
				continue
			}
			var localPort int
			sLocalPort := s[0]
			if sLocalPort == "" {
				// get random port in local
				localPort, err = ports.GetAvailablePort()
				if err != nil {
					fmt.Printf("failed to get local port: %s \n", err)
					continue
				}
			}
			if sLocalPort != "" {
				localPort, err = strconv.Atoi(sLocalPort)
				if err != nil {
					fmt.Printf("wrong format of local port:%d , skipped. \n", localPort)
					continue
				}
			}
			fmt.Printf("remote port convert before: %s \n", s[1])
			remotePort, err := strconv.Atoi(s[1])
			if err != nil {
				fmt.Printf("wrong format of remote port:%d , err: %s, skipped. \n", remotePort, err.Error())
				continue
			}
			localPorts = append(localPorts, localPort)
			remotePorts = append(remotePorts, remotePort)
		}
		fmt.Printf("ready to call dev port forward locals: %d, remotes: %d \n", localPorts, remotePorts)
		// listening, it will wait until kill port forward progress
		if len(localPorts) > 0 && len(remotePorts) > 0 {
			nocalhostApp.PortForwardInBackGround(deployment, podName, podNameSpace, localPorts, remotePorts)
		}
		fmt.Print("not needed port forward")
	},
}
