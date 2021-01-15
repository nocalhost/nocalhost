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
			log.Fatalf("Service \"%s\" is not in DevMode", deployment)
		}

		if nocalhostApp.CheckIfSvcIsPortForwarded(deployment) {
			log.Fatalf("Service \"%s\" is already in port-forwarding", deployment)
		}

		// look for nhctl
		nhctlAbsdir, err := exec.LookPath(nocalhostApp.GetMyBinName())
		if err != nil {
			log.Fatal("Nhctl not found")
		}
		// overwrite Args[0] as ABS directory of bin directory
		os.Args[0] = nhctlAbsdir

		if portForwardOptions.RunAsDaemon {
			//log.Infof("Running port-forward in background, parent pid is %d, ppid is %d", os.Getpid(), os.Getppid())
			_, err := daemon.Background(nocalhostApp.GetPortForwardLogFile(deployment), nocalhostApp.GetApplicationBackGroundOnlyPortForwardPidFile(deployment), true)
			if err != nil {
				log.Fatal("Failed to run port-forward background, please try again")
			}
		}

		log.Info("Start port-forwarding")

		// find deployment pods
		podName, err := nocalhostApp.GetNocalhostDevContainerPod(deployment)
		if err != nil {
			log.FatalE(err, "No dev container found")
		}

		// run in child process
		if len(portForwardOptions.DevPort) == 0 {
			// if not specify -p then use default
			portForwardOptions.DevPort = nocalhostApp.GetDefaultDevPort(deployment)
			log.Debugf("Get default devPort: %s ", portForwardOptions.DevPort)
		}
		var localPorts []int
		var remotePorts []int
		for _, port := range portForwardOptions.DevPort {
			// 8080:8080, :8080
			s := strings.Split(port, ":")
			if len(s) < 2 {
				// ignore wrong format
				log.Warnf("Wrong format of dev port:%s , skipped.", port)
				continue
			}
			var localPort int
			sLocalPort := s[0]
			if sLocalPort == "" {
				// get random port in local
				localPort, err = ports.GetAvailablePort()
				if err != nil {
					log.WarnE(err, "Failed to get local port")
					continue
				}
			} else {
				localPort, err = strconv.Atoi(sLocalPort)
				if err != nil {
					log.Warnf("Wrong format of local port:%s , skipped.", sLocalPort)
					continue
				}
			}
			remotePort, err := strconv.Atoi(s[1])
			if err != nil {
				log.ErrorE(err, fmt.Sprintf("wrong format of remote port: %s, skipped", s[1]))
				continue
			}
			localPorts = append(localPorts, localPort)
			remotePorts = append(remotePorts, remotePort)
		}
		log.Infof("Ready to call dev port forward locals: %d, remotes: %d", localPorts, remotePorts)
		// listening, it will wait until kill port forward progress
		if len(localPorts) > 0 && len(remotePorts) > 0 {
			nocalhostApp.PortForwardInBackGround(deployment, podName, localPorts, remotePorts)
		}
		log.Info("No need to port forward")
	},
}
