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
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/request"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"strconv"
	"strings"
	"time"
)

type Init struct {
	Type      string
	Port      int
	NameSpace string
	Set       []string
}

var inits = &Init{}

func init() {
	InitCommand.Flags().StringVarP(&inits.Type, "type", "t", "", "set NodePort or LoadBalancer to expose nocalhost service")
	InitCommand.Flags().IntVarP(&inits.Port, "port", "p", 0, "for NodePort usage set ports")
	InitCommand.Flags().StringVarP(&inits.NameSpace, "namespace", "n", "default", "set init nocalhost namesapce")
	InitCommand.Flags().StringSliceVar(&inits.Set, "set", []string{}, "set values of helm")
	rootCmd.AddCommand(InitCommand)
}

var InitCommand = &cobra.Command{
	Use:   "init",
	Short: "Init application",
	Long:  "Init api, web and dep component in you cluster",
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		kubectl, err := tools.CheckThirdPartyCLI()
		if err != nil {
			log.Fatalf("%s, you should install them first(helm3 and kubectl)", err.Error())
		}
		// init api and web
		// nhctl install nocalhost -u https://e.coding.net/codingcorp/nocalhost/nocalhost.git -t helm --kubeconfig xxx -n xxx
		params := []string{
			"install",
			"nocalhost",
			"-u",
			app.DefaultInitHelmGitRepo,
			"--kubeconfig",
			settings.KubeConfig,
			"-n",
			inits.NameSpace,
			"--type",
			app.DefaultInitHelmType,
			"--resource-path",
			app.DefaultInitHelmResourcePath,
		}
		if inits.Type != "" {
			if strings.ToLower(inits.Type) == "nodeport" {
				inits.Type = "NodePort"
				if inits.Port == 0 {
					// random Port
					// By default, minikube only exposes ports 30000-32767
					inits.Port = tools.GenerateRangeNum(30000, 32767)
					params = append(params, "--set", "service.port="+strconv.Itoa(inits.Port))
				}
			}
			if strings.ToLower(inits.Type) == "loadbalancer" {
				inits.Type = "LoadBalancer"
			}
			params = append(params, "--set", "service.type="+inits.Type)
		}
		if len(inits.Set) > 0 {
			for _, set := range inits.Set {
				params = append(params, "--set", set)
			}
		}
		client, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, app.DefaultClientGoTimeOut)
		if err != nil || client == nil {
			log.Fatalf("new go client fail, err %s, or check you kubeconfig\n", err)
		}
		// check if exist namespace
		err = client.CheckExistNameSpace(inits.NameSpace)
		if err != nil {
			err = client.CreateNameSpace(inits.NameSpace)
			if err != nil {
				log.Fatalf("init fail, create namespace %s fail, err: %s\n", inits.NameSpace, err.Error())
			}
		}

		// call install command

		nhctl := tools.GetNhctl()
		_, err = tools.ExecCommand(nil, true, nhctl, params...)
		if err != nil {
			log.Fatalf("execution nhctl install fail %s, try run nhctl uninstall nocalhost --force\n", err.Error())
		}

		// 1. watch nocalhost-api and nocalhost-web ready
		// 2. print nocalhost-web service address
		// 3. use nocalhost-web service address to set default data into cluster
		spinner := utils.NewSpinner(" waiting for nocalhost component ready, this will take a few minutes...")
		spinner.Start()
		err = client.WaitDeploymentToBeReady(inits.NameSpace, app.DefaultInitWatchDeployment, app.DefaultClientGoTimeOut)
		if err != nil {
			log.Fatalf("watch deployment %s timeout, err: %s\n", app.DefaultInitWatchDeployment, err.Error())
		}
		// wait nocalhost-web ready
		// max 5 min
		checkTime := 0
		for {
			isReady, _ := client.CheckDeploymentReady(context.TODO(), inits.NameSpace, app.DefaultInitWatchWebDeployment)
			if isReady {
				break
			}
			checkTime = checkTime + 1
			if checkTime > 1500 {
				break
			}
			time.Sleep(time.Duration(200) * time.Millisecond)
		}
		spinner.Stop()

		// get Node ExternalIP
		nodes, err := client.GetNodesList()
		if err != nil {
			log.Fatalf("get nodes fail, err %s\n", err)
		}

		nodeExternalIP := ""
		nodeInternalIP := ""
		loadBalancerIP := ""
		for _, node := range nodes.Items {
			done := false
			for _, address := range node.Status.Addresses {
				if address.Type == corev1.NodeExternalIP {
					nodeExternalIP = address.Address
					done = true
					break
				}
				if address.Type == corev1.NodeInternalIP {
					nodeInternalIP = address.Address
				}
			}
			if done {
				break
			}
		}

		// get loadbalancer service IP
		service, err := client.GetService(app.DefaultInitNocalhostService, inits.NameSpace)
		if err != nil {
			log.Fatalf("get service %s fail, please try again\n", err)
		}
		for _, ip := range service.Status.LoadBalancer.Ingress {
			if ip.IP != "" {
				loadBalancerIP = ip.IP
				break
			}
		}
		// should use loadbalancerIP > nodeExternalIP > nodeInternalIP
		// fmt.Printf("%s %s %s", loadBalancerIP, nodeExternalIP, nodeInternalIP)
		endPoint := ""
		if nodeInternalIP != "" {
			endPoint = nodeInternalIP + ":" + strconv.Itoa(inits.Port)
		}
		if nodeExternalIP != "" {
			endPoint = nodeExternalIP + ":" + strconv.Itoa(inits.Port)
		}
		if loadBalancerIP != "" {
			endPoint = loadBalancerIP
		}
		fmt.Printf("Nocalhost get ready, endpoint is: %s \n", endPoint)

		// set default cluster, application, users
		req := request.NewReq(fmt.Sprintf("http://%s", endPoint), settings.KubeConfig, kubectl, inits.NameSpace)
		kubeResult := req.CheckIfMiniKube().Login(app.DefaultInitAdminUserName, app.DefaultInitAdminPassWord).GetKubeConfig().AddBookInfoApplication("").AddCluster().AddUser(app.DefaultInitUserEmail, app.DefaultInitPassword, app.DefaultInitName).AddDevSpace()
		// wait for nocalhost-dep deployment in nocalhost-reserved namespace
		spinner = utils.NewSpinner(" waiting for nocalhost-dep ready, this will take a few minutes...")
		spinner.Start()
		err = client.WaitDeploymentToBeReady(app.DefaultInitWaitNameSpace, app.DefaultInitWaitDeployment, app.DefaultClientGoTimeOut)
		if err != nil {
			log.Fatalf("watch deployment %s timeout, err: %s\n", app.DefaultInitWatchDeployment, err.Error())
		}
		spinner.Stop()
		if kubeResult.Minikube {
			portResult := req.GetAvailableRandomLocalPort()
			serverUrl := fmt.Sprintf("%s:%d", "127.0.0.1", portResult.MiniKubeAvailablePort)
			coloredoutput.Success("nocalhost init completed. \n Server Url: %s \n Username: %s \n Password: %s \n please set VS Code Plugin and login, enjoy! \n port forwarding, please do not close this windows! \n", serverUrl, app.DefaultInitUserEmail, app.DefaultInitPassword)
			req.RunPortForward(portResult.MiniKubeAvailablePort)
		} else {
			coloredoutput.Success("nocalhost init completed. \n Server Url: %s \n Username: %s \n Password: %s \n please set VS Code Plugin and login, enjoy! \n", fmt.Sprintf("http://%s", endPoint), app.DefaultInitUserEmail, app.DefaultInitPassword)
		}
	},
}
