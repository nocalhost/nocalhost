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
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Init struct {
	Type                   string
	Port                   int
	NameSpace              string
	Set                    []string
	Source                 string
	Force                  bool
	InjectUserTemplate     string
	InjectUserAmount       int
	InjectUserAmountOffset int
}

var inits = &Init{}

func init() {
	InitCommand.Flags().StringVarP(&inits.Type, "type", "t", "", "set NodePort or LoadBalancer to expose nocalhost service")
	InitCommand.Flags().IntVarP(&inits.Port, "port", "p", 80, "for NodePort usage set ports")
	InitCommand.Flags().StringVarP(&inits.Source, "source", "s", "", "bookinfo source, github or coding, default is github")
	InitCommand.Flags().StringVarP(&inits.NameSpace, "namespace", "n", "nocalhost", "set init nocalhost namesapce")
	InitCommand.Flags().StringSliceVar(&inits.Set, "set", []string{}, "set values of helm")
	InitCommand.Flags().BoolVar(&inits.Force, "force", false, "force to init, warning: it will remove all nocalhost old data")
	InitCommand.Flags().StringVar(&inits.InjectUserTemplate, "inject-user-template", "", "inject users template, example Techo%d, max length is 15")
	InitCommand.Flags().IntVar(&inits.InjectUserAmount, "inject-user-amount", 0, "inject user amount, example 10, max is 999")
	InitCommand.Flags().IntVar(&inits.InjectUserAmountOffset, "inject-user-offset", 1, "inject user id offset, default is 1")
	rootCmd.AddCommand(InitCommand)
}

var InitCommand = &cobra.Command{
	Use:   "init",
	Short: "Init application",
	Long:  "Init api, web and dep component in you cluster",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(inits.InjectUserTemplate) > 15 {
			log.Fatal("--inject-user-template length should less then 15")
		}
		if inits.InjectUserTemplate != "" && !strings.ContainsAny(inits.InjectUserTemplate, "%d") {
			log.Fatal("--inject-user-template does not contains %d")
		}
		if inits.InjectUserAmount > 999 {
			log.Fatal("--inject-user-amount must less then 999")
		}
		if (len(strconv.Itoa(inits.InjectUserAmountOffset)) + len(inits.InjectUserTemplate)) > 20 {
			log.Fatal("--inject-user-offset and --inject-user-template length can not greater than 20")
		}
		switch inits.NameSpace {
		case "default":
			log.Fatal("please do not init nocalhost in default namespace")
		case "kube-system":
			log.Fatal("please do not init nocalhost in kube-system namespace")
		case "kube-public":
			log.Fatal("please do not init nocalhost in kube-public namespace")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		kubectl, err := tools.CheckThirdPartyCLI()
		if err != nil {
			log.Fatalf("%s, you should install them first(helm3 and kubectl)", err.Error())
		}
		if settings.KubeConfig == "" {
			u, err := user.Current()
			if err == nil {
				settings.KubeConfig = filepath.Join(u.HomeDir, ".kube", "config")
			}
		}
		// init api and web
		// nhctl install nocalhost -u https://e.coding.net/codingcorp/nocalhost/nocalhost.git -t helm --kubeconfig xxx -n xxx
		nocalhostHelmSource := app.DefaultInitHelmGitRepo
		if strings.ToLower(inits.Source) == "coding" {
			nocalhostHelmSource = app.DefaultInitHelmCODINGGitRepo
		}

		params := []string{
			"install",
			app.DefaultInitInstallApplicationName,
			"-u",
			nocalhostHelmSource,
			"--kubeconfig",
			settings.KubeConfig,
			"-n",
			inits.NameSpace,
			"--type",
			app.DefaultInitHelmType,
			"--resource-path",
			app.DefaultInitHelmResourcePath,
		}
		// set install api and web image version
		setComponentDockerImageVersion(&params)

		if inits.Type != "" {
			if strings.ToLower(inits.Type) == "nodeport" {
				inits.Type = "NodePort"
			}
			if strings.ToLower(inits.Type) == "loadbalancer" {
				inits.Type = "LoadBalancer"
			}
			params = append(params, "--set", "service.type="+inits.Type)
		}

		// add ports, default is 80
		params = append(params, "--set", "service.port="+strconv.Itoa(inits.Port))

		if len(inits.Set) > 0 {
			for _, set := range inits.Set {
				params = append(params, "--set", set)
			}
		}
		client, err := clientgoutils.NewClientGoUtils(settings.KubeConfig, app.DefaultClientGoTimeOut)
		fmt.Printf("kubeconfig %s \n", settings.KubeConfig)
		if err != nil || client == nil {
			log.Fatalf("new go client fail, err %s, or check you kubeconfig\n", err)
		}

		nhctl := tools.GetNhctl()
		// if force init, remove all init data first
		if inits.Force {
			spinner := utils.NewSpinner(" waiting for delete old data, this will take a few minutes...")
			spinner.Start()
			uninstall := []string{
				"uninstall",
				app.DefaultInitInstallApplicationName,
				"--force",
			}
			_, err = tools.ExecCommand(nil, true, nhctl, uninstall...)
			if err != nil {
				log.Warnf("uninstall %s application fail, ignore", app.DefaultInitInstallApplicationName)
			}
			// delete nocalhost(server namespace), nocalhost-reserved(dep) namespace if exist
			if nsErr := client.CheckExistNameSpace(inits.NameSpace); nsErr == nil {
				err := client.DeleteNameSpace(inits.NameSpace, true)
				if err != nil {
					log.Warnf("delete namespace %s fail, ignore", inits.NameSpace)
				}
			}
			if nsErr := client.CheckExistNameSpace(app.DefaultInitWaitNameSpace); nsErr == nil {
				err = client.DeleteNameSpace(app.DefaultInitWaitNameSpace, true)
				if err != nil {
					log.Warnf("delete namespace %s fail, ignore", app.DefaultInitWaitNameSpace)
				}
			}
			spinner.Stop()
		}

		// normal init: check if exist namespace
		err = client.CheckExistNameSpace(inits.NameSpace)
		if err != nil {
			customLabels := map[string]string{
				"env": app.DefaultInitCreateNameSpaceLabels,
			}
			err = client.CreateNameSpace(inits.NameSpace, customLabels)
			if err != nil {
				log.Fatalf("init fail, create namespace %s fail, err: %s\n", inits.NameSpace, err.Error())
			}
		}
		// call install command
		_, err = tools.ExecCommand(nil, true, nhctl, params...)
		if err != nil {
			coloredoutput.Fail("execution nhctl install fail %s, try to add `--force` end of command manually\n", err.Error())
			log.Fatal("exit init")
		}

		// 1. watch nocalhost-api and nocalhost-web ready
		// 2. print nocalhost-web service address
		// 3. use nocalhost-web service address to set default data into cluster
		spinner := utils.NewSpinner(" waiting for Nocalhost component ready, this will take a few minutes...")
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
			if inits.Port != 80 {
				endPoint = loadBalancerIP + ":" + strconv.Itoa(inits.Port)
			}
		}
		fmt.Printf("Nocalhost get ready, endpoint is: %s \n", endPoint)

		// bookinfo source from
		source := app.DefaultInitApplicationGithub
		if strings.ToLower(inits.Source) == "coding" {
			source = app.DefaultInitApplicationCODING
		}

		// set default cluster, application, users
		req := request.NewReq(fmt.Sprintf("http://%s", endPoint), settings.KubeConfig, kubectl, inits.NameSpace, inits.Port)
		kubeResult := req.CheckIfMiniKube().Login(app.DefaultInitAdminUserName, app.DefaultInitAdminPassWord).GetKubeConfig().AddBookInfoApplication(source).AddCluster().AddUser(app.DefaultInitUserEmail, app.DefaultInitPassword, app.DefaultInitName).AddDevSpace()
		// should inject batch user
		if inits.InjectUserTemplate != "" && inits.InjectUserAmount > 0 {
			_ = req.SetInjectBatchUserTemplate(inits.InjectUserTemplate).InjectBatchDevSpace(inits.InjectUserAmount, inits.InjectUserAmountOffset)
		}
		// change dep images tag
		setDepComponentDockerImage(kubectl, settings.KubeConfig)

		// wait for nocalhost-dep deployment in nocalhost-reserved namespace
		spinner = utils.NewSpinner(" waiting for Nocalhost-dep ready, this will take a few minutes...")
		spinner.Start()
		err = client.WaitDeploymentToBeReady(app.DefaultInitWaitNameSpace, app.DefaultInitWaitDeployment, app.DefaultClientGoTimeOut)
		if err != nil {
			log.Fatalf("watch deployment %s timeout, err: %s\n", app.DefaultInitWatchDeployment, err.Error())
		}
		spinner.Stop()
		if kubeResult.Minikube {
			// use default DefaultInitMiniKubePortForwardPort port-forward
			//portResult := req.GetAvailableRandomLocalPort()
			port := app.DefaultInitMiniKubePortForwardPort
			if !req.CheckPortIsAvailable(app.DefaultInitMiniKubePortForwardPort) {
				port = req.GetAvailableRandomLocalPort().MiniKubeAvailablePort
			}
			serverUrl := fmt.Sprintf("http://%s:%d", "127.0.0.1", port)
			coloredoutput.Success("Nocalhost init completed. \n Server Url: %s \n Plugin User: \n Username: %s \n Password: %s \n Admin User (Web UI): \n Username: %s \n Password: %s \n please setup VS Code Plugin and login, enjoy! \n", serverUrl, app.DefaultInitUserEmail, app.DefaultInitPassword, app.DefaultInitAdminUserName, app.DefaultInitAdminPassWord)
			coloredoutput.Information("port forwarding, please do not close this windows! \n")
			// if DefaultInitMiniKubePortForwardPort can not use, it will return available port
			req.RunPortForward(port)
		} else {
			serverUrl := fmt.Sprintf("http://%s", endPoint)
			coloredoutput.Success("Nocalhost init completed. \n Server Url: %s \n Plugin User: \n Username: %s \n Password: %s \n Admin User (Web UI): \n Username: %s \n Password: %s \n please setup VS Code Plugin and login, enjoy! \n", serverUrl, app.DefaultInitUserEmail, app.DefaultInitPassword, app.DefaultInitAdminUserName, app.DefaultInitAdminPassWord)
		}
	},
}

func setComponentDockerImageVersion(params *[]string) {
	if Branch == "" {
		return
	}
	// set -r with nhctl install
	// version is nocalhost tag
	gitRef := Version
	if Branch != app.DefaultNocalhostMainBranch {
		gitRef = Branch
	}
	*params = append(*params, "-r", gitRef)
	// main branch, means use version for docker images
	// Branch will set by make nhctl
	if Branch == app.DefaultNocalhostMainBranch {
		log.Infof("Init nocalhost component with release %s", Version)
		*params = append(*params, "--set", "api.image.tag="+Version)
		*params = append(*params, "--set", "web.image.tag="+Version)
	} else {
		log.Infof("Init nocalhost component with dev %s, but nocalhost-web with dev tag only", DevGitCommit)
		*params = append(*params, "--set", "api.image.tag="+DevGitCommit)
		// because of web image and api has different commitID, so take latest dev tag
		*params = append(*params, "--set", "web.image.tag=dev")
	}
}

// because of dep run latest docker image, so it can only use kubectl set image to set dep docker version same as nhctl version
func setDepComponentDockerImage(kubectl, kubeConfig string) {
	if Branch == "" {
		return
	}
	tag := Version
	// Branch will set by make nhctl
	if Branch == app.DefaultNocalhostMainBranch {
		tag = DevGitCommit
	}
	params := []string{
		"set",
		"image",
		"deployment/" + app.DefaultInitWaitDeployment,
		fmt.Sprintf("%s=%s:%s", app.DefaultInitWaitDeployment, app.DefaultNocalhostDepDockerRegistry, tag),
		"-n",
		app.DefaultInitWaitNameSpace,
		"--kubeconfig",
		kubeConfig,
	}
	_, err := tools.ExecCommand(nil, true, kubectl, params...)
	if err != nil {
		log.Warnf("set nocalhost-dep component tag fail, err: %s", err)
	}
}
