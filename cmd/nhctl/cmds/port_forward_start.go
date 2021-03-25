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
	"strconv"
	"strings"
)

var portForwardOptions = &app.PortForwardOptions{}

func init() {
	portForwardStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	portForwardStartCmd.Flags().StringSliceVarP(&portForwardOptions.DevPort, "dev-port", "p", []string{}, "port-forward between pod and local, such 8080:8080 or :8080(random localPort)")
	//portForwardStartCmd.Flags().BoolVarP(&portForwardOptions.RunAsDaemon, "daemon", "m", true, "if port-forward run as daemon")
	portForwardStartCmd.Flags().BoolVarP(&portForwardOptions.Forward, "forward", "f", false, "forward actually, deprecated")
	portForwardStartCmd.Flags().StringVarP(&portForwardOptions.PodName, "pod", "", "", "specify pod name")
	portForwardStartCmd.Flags().StringVarP(&container, "container", "c", "", "which container of pod to run command")
	portForwardStartCmd.Flags().StringVarP(&portForwardOptions.ServiceType, "type", "", "deployment", "specify service type")
	portForwardStartCmd.Flags().StringVarP(&portForwardOptions.Way, "way", "", "manual", "specify port-forward way, deprecated")
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
		initAppAndCheckIfSvcExist(applicationName, deployment, []string{portForwardOptions.ServiceType})

		log.Info("Starting port-forwarding")

		// find deployment pods
		podName, err := nocalhostApp.GetNocalhostDevContainerPod(deployment)
		if err != nil {
			// use serviceType get pods name
			// can not find devContainer, means need port-forward normal service, get pods from command flags
			podName = portForwardOptions.PodName
		}

		if len(portForwardOptions.DevPort) == 0 {
			// if not specify -p then use default
			portForwardOptions.DevPort = nocalhostApp.GetDefaultDevPort(deployment, container)
			log.Debugf("Get default devPort: %s ", portForwardOptions.DevPort)
		}
		var localPorts []int
		var remotePorts []int
		for _, port := range portForwardOptions.DevPort {
			localPort, remotePort, err := getPortForwardForString(port)
			if err != nil {
				log.WarnE(err, "")
				continue
			}
			localPorts = append(localPorts, localPort)
			remotePorts = append(remotePorts, remotePort)
		}

		for index, localPort := range localPorts {
			if err = nocalhostApp.PortForward(deployment, podName, localPort, remotePorts[index]); err != nil {
				log.FatalE(err, "")
			}
		}
	},
}

// portStr is like 8080:80 or :80
func getPortForwardForString(portStr string) (int, int, error) {
	var err error
	s := strings.Split(portStr, ":")
	if len(s) < 2 {
		return 0, 0, errors.New(fmt.Sprintf("Wrong format of port: %s", portStr))
	}
	var localPort, remotePort int
	sLocalPort := s[0]
	if sLocalPort == "" {
		// get random port in local
		if localPort, err = ports.GetAvailablePort(); err != nil {
			return 0, 0, err
		}
	} else if localPort, err = strconv.Atoi(sLocalPort); err != nil {
		return 0, 0, errors.Wrap(err, fmt.Sprintf("Wrong format of local port: %s.", sLocalPort))
	}
	if remotePort, err = strconv.Atoi(s[1]); err != nil {
		return 0, 0, errors.Wrap(err, fmt.Sprintf("wrong format of remote port: %s, skipped", s[1]))
	}
	return localPort, remotePort, nil
}
