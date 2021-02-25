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
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var portForwardOptions = &app.PortForwardOptions{}

func init() {
	portForwardStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	portForwardStartCmd.Flags().StringSliceVarP(&portForwardOptions.DevPort, "dev-port", "p", []string{}, "port-forward between pod and local, such 8080:8080 or :8080(random localPort)")
	//portForwardStartCmd.Flags().BoolVarP(&portForwardOptions.RunAsDaemon, "daemon", "m", true, "if port-forward run as daemon")
	portForwardStartCmd.Flags().BoolVarP(&portForwardOptions.Forward, "forward", "f", false, "forward actually")
	portForwardStartCmd.Flags().StringVarP(&portForwardOptions.PodName, "pod", "", "", "specify pod name")
	portForwardStartCmd.Flags().StringVarP(&portForwardOptions.Way, "way", "", "manual", "specify port-forward way")
	PortForwardCmd.AddCommand(portForwardStartCmd)
}

var portForwardStartCmd = &cobra.Command{
	Use:   "start [NAME]",
	Short: "Forward local port to remote pod's port",
	Long:  `Forward local port to remote pod's port`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		if portForwardOptions.Way != "manual" {
			portForwardOptions.Way = "devPorts"
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		applicationName := args[0]
		InitAppAndCheckIfSvcExist(applicationName, deployment)

		// look for nhctl
		nhctlAbsdir, err := exec.LookPath(nocalhostApp.GetMyBinName())
		if err != nil {
			log.Fatal("nhctl command not found")
		}
		// overwrite Args[0] as ABS directory of bin directory
		os.Args[0] = nhctlAbsdir

		log.Info("Starting port-forwarding")

		// find deployment pods
		podName, err := nocalhostApp.GetNocalhostDevContainerPod(deployment)
		if err != nil {
			// can not find devContainer, get pods from command flags
			podName = portForwardOptions.PodName
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
		// change -p flag os.Args
		nocalhostApp.FixPortForwardOSArgs(localPorts, remotePorts)

		// listening, it will wait until kill port forward progress
		listenAddress := []string{"0.0.0.0"}
		if len(localPorts) > 0 && len(remotePorts) > 0 {
			nocalhostApp.PortForwardInBackGround(listenAddress, deployment, podName, localPorts, remotePorts, portForwardOptions.Way, portForwardOptions.Forward)
		}

		log.Info("No need to port forward")
	},
}
