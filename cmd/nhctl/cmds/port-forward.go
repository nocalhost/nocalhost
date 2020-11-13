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
	"math/rand"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/third_party/kubectl"
	"nocalhost/pkg/nhctl/utils"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var remotePort string
var app *Application

type PortForwardFlags struct {
	*EnvSettings
	LocalPort  int
	RemotePort int
	Deployment string
}

var portForwardFlags = PortForwardFlags{
	EnvSettings: settings,
}

func init() {
	portForwardCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	//portForwardCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubernetes cluster config")
	portForwardCmd.Flags().IntVarP(&portForwardFlags.LocalPort, "local-port", "l", 0, "local port to forward")
	portForwardCmd.Flags().IntVarP(&portForwardFlags.RemotePort, "remote-port", "r", 0, "remote port to be forwarded")
	portForwardCmd.Flags().StringVarP(&portForwardFlags.Deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
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
		app, err = NewApplication(applicationName)
		clientgoutils.Must(err)
		//nocalhostConfig = app.Config
		if portForwardFlags.Deployment == "" {
			fmt.Println("error: please use -d to specify a kubernetes deployment")
			return
		}
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}

		svcConfig := app.Config.GetSvcConfig(portForwardFlags.Deployment)
		var configLocalPort, configRemotePort int
		//if svcConfig != nil && svcConfig.SshPort != nil {
		//	configLocalPort, err = strconv.Atoi(strings.Trim(strings.Split(svcConfig.DevPort[0], ":")[0], " "))
		//	utils.Mush(err)
		//	configRemotePort, err = strconv.Atoi(strings.Trim(strings.Split(svcConfig.DevPort[0], ":")[1], " "))
		//	utils.Mush(err)
		//}
		if svcConfig != nil && svcConfig.SshPort != nil {
			configLocalPort = svcConfig.SshPort.LocalPort
			configRemotePort = svcConfig.SshPort.SshPort
		}

		if portForwardFlags.LocalPort == 0 {
			if configLocalPort != 0 {
				portForwardFlags.LocalPort = configLocalPort
			} else {
				// generate a random local port
				rand.Seed(time.Now().Unix())
				portForwardFlags.LocalPort = rand.Intn(10000) + 30001
				debug("local port not specify, use random port : %s", portForwardFlags.LocalPort)
			}
		}

		if portForwardFlags.RemotePort == 0 {
			if configRemotePort != 0 {
				portForwardFlags.RemotePort = configRemotePort
			} else {
				portForwardFlags.RemotePort = DefaultForwardRemotePort
				debug("remote port not specify, use default port : %d", portForwardFlags.RemotePort)
			}
		}

		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT) // kill -1
		ctx, cancel := context.WithCancel(context.TODO())

		go func() {
			<-c
			cancel()
			fmt.Println("stop port forward")
			CleanupPid()
		}()

		// todo check if there is a same port-forward exists

		pid := os.Getpid()
		pidDir := app.GetPortForwardPidDir(pid)
		utils.Mush(os.Mkdir(pidDir, 0755))

		debug("recording port-forward info...")
		clientgoutils.Must(app.SavePortForwardInfo(portForwardFlags.LocalPort, portForwardFlags.RemotePort))
		err = kubectl.PortForward(ctx, settings.KubeConfig, nameSpace, portForwardFlags.Deployment, fmt.Sprintf("%d", portForwardFlags.LocalPort), fmt.Sprintf("%d", portForwardFlags.RemotePort)) // eg : ./utils/darwin/kubectl port-forward --address 0.0.0.0 deployment/coding  12345:22
		if err != nil {
			fmt.Printf("failed to forward port : %v\n", err)
			CleanupPid()
		}
	},
}

func CleanupPid() {
	pidDir := app.GetPortForwardPidDir(os.Getpid())
	if _, err2 := os.Stat(pidDir); err2 != nil {
		if os.IsNotExist(err2) {
			debug("%s not exits, no need to cleanup it", pidDir)
			return
		} else {
			fmt.Printf("[warning] fails to cleanup %s\n", pidDir)
		}
	}
	err := os.RemoveAll(pidDir)
	if err != nil {
		fmt.Printf("removing .pid failed, please remove it manually, err:%v\n", err)
	} else {
		debug("%s cleanup", pidDir)
	}
}
