/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"nocalhost/cmd/nhctl/cmds/common"
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

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
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
	InitCommand.Flags().StringVarP(
		&inits.Type, "type", "t", "",
		"set NodePort or LoadBalancer to expose nocalhost service",
	)
	InitCommand.Flags().IntVarP(
		&inits.Port, "port", "p", 80,
		"for NodePort usage set ports",
	)
	InitCommand.Flags().StringVarP(
		&inits.Source, "source", "s", "",
		"(Deprecated) bookinfo source, github or coding, default is github",
	)
	InitCommand.Flags().StringVarP(
		&inits.NameSpace, "namespace", "n",
		"nocalhost", "set init nocalhost namesapce",
	)
	InitCommand.Flags().StringSliceVar(
		&inits.Set, "set", []string{},
		"set values of helm",
	)
	InitCommand.Flags().BoolVar(
		&inits.Force, "force", false,
		"force to init, warning: it will remove all nocalhost old data",
	)
	InitCommand.Flags().StringVar(
		&inits.InjectUserTemplate, "inject-user-template", "",
		"inject users template, example Techo%d, max length is 15",
	)
	InitCommand.Flags().IntVar(
		&inits.InjectUserAmount, "inject-user-amount", 0,
		"inject user amount, example 10, max is 999",
	)
	InitCommand.Flags().IntVar(
		&inits.InjectUserAmountOffset, "inject-user-offset", 1,
		"inject user id offset, default is 1",
	)
	InitCmd.AddCommand(InitCommand)
}

var InitCommand = &cobra.Command{
	Use:   "demo",
	Short: "Init Nocalhost with demo mode",
	Long:  "Init api, web and dep component in cluster",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(inits.InjectUserTemplate) > 15 {
			log.Fatal("--inject-user-template length should less then 15")
		}
		if inits.InjectUserTemplate != "" && !strings.ContainsAny(inits.InjectUserTemplate, "%d") {
			log.Fatalf("--inject-user-template does not contains %s", inits.InjectUserTemplate)
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
		mustI(err, "you should install them first(helm3 and kubectl)")
		must(common.Prepare())

		// init api and web
		// nhctl install nocalhost -u https://e.coding.net/nocalhost/nocalhost/nocalhost.git
		// -t helm --kubeconfig xxx -n xxx
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
			common.KubeConfig,
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
		client, err := clientgoutils.NewClientGoUtils(common.KubeConfig, inits.NameSpace)
		log.Debugf("kubeconfig %s \n", common.KubeConfig)
		if err != nil || client == nil {
			log.Fatalf("new go client fail, err %s, or check you kubeconfig\n", err)
			return
		}

		nhctl, err := utils.GetNhctlPath()
		if err != nil {
			log.FatalE(err, "")
			return
		}
		// if force init, remove all init data first
		if inits.Force {
			spinner := utils.NewSpinner(" waiting for force uninstall Nocalhost...")
			spinner.Start()
			uninstall := []string{
				"uninstall",
				app.DefaultInitInstallApplicationName,
				"--force",
				"-n",
				inits.NameSpace,
			}
			_, err = tools.ExecCommand(nil, debug, false, true, nhctl, uninstall...)
			utils.ShouldI(err, fmt.Sprintf("uninstall %s application fail", app.DefaultInitInstallApplicationName))
			// delete nocalhost(server namespace), nocalhost-reserved(dep) namespace if exist
			if nsErr := client.CheckExistNameSpace(inits.NameSpace); nsErr == nil {
				// try delete mariadb
				_ = client.NameSpace(inits.NameSpace).DeleteStatefulSetAndPVC("nocalhost-mariadb")
				utils.ShouldI(
					client.DeleteNameSpace(inits.NameSpace, true),
					fmt.Sprintf("delete namespace %s faile", inits.NameSpace),
				)
			}
			if nsErr := client.CheckExistNameSpace(app.DefaultInitWaitNameSpace); nsErr == nil {
				err = client.DeleteNameSpace(app.DefaultInitWaitNameSpace, true)
				utils.ShouldI(err, fmt.Sprintf("delete namespace %s fail", app.DefaultInitWaitNameSpace))
			}
			spinner.Stop()
			coloredoutput.Success("force uninstall Nocalhost successfully \n")
		}

		log.Debugf("checking namespace %s if exist", inits.NameSpace)
		// normal init: check if exist namespace
		err = client.CheckExistNameSpace(inits.NameSpace)
		if err != nil {
			customLabels := map[string]string{
				"env": app.DefaultInitCreateNameSpaceLabels,
			}
			mustI(client.Labels(customLabels).CreateNameSpace(inits.NameSpace), "create namespace fail")
		}
		spinner := utils.NewSpinner(" waiting for get Nocalhost manifest...")
		spinner.Start()
		// call install command
		_, err = tools.ExecCommand(nil, debug, true, false, nhctl, params...)
		if err != nil {
			coloredoutput.Fail(
				"\n nhctl init fail, try to add `--force` end of command manually\n",
			)
			log.Fatal("exit init")
		}
		spinner.Stop()

		// 1. watch nocalhost-api and nocalhost-web ready
		// 2. print nocalhost-web service address
		// 3. use nocalhost-web service address to set default data into cluster
		spinner = utils.NewSpinner(" waiting for Nocalhost component ready, this will take a few minutes...")
		spinner.Start()
		mustI(
			client.NameSpace(inits.NameSpace).WaitDeploymentToBeReady(app.DefaultInitWatchDeployment),
			"watch deployment timeout",
		)
		// wait nocalhost-web ready
		// max 5 min
		checkTime := 0
		for {
			isReady, _ := client.NameSpace(inits.NameSpace).CheckDeploymentReady(app.DefaultInitWatchWebDeployment)
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
		coloredoutput.Success("Nocalhost component get ready \n")
		// bookinfo source from
		//source := app.DefaultInitApplicationGithub
		//if strings.ToLower(inits.Source) == "coding" {
		//	source = app.DefaultInitApplicationCODING
		//}

		spinner = utils.NewSpinner(" waiting for init demo data...")
		spinner.Start()

		log.Debugf("try to find out web endpoint")
		endpoint := FindOutWebEndpoint(client)

		log.Debugf("try login and init nocalhost web(User、DevSpace and demo applications)")
		// set default cluster, application, users
		req := request.NewReq(
			fmt.Sprintf("http://%s", endpoint), common.KubeConfig, kubectl, inits.NameSpace, inits.Port,
		).Login(
			app.DefaultInitAdminUserName, app.DefaultInitAdminPassWord,
		).GetKubeConfig().AddBookInfoApplicationForThree().AddCluster().AddUser(
			app.DefaultInitUserEmail, app.DefaultInitPassword, app.DefaultInitName,
		).AddDevSpace()

		// should inject batch user
		if inits.InjectUserTemplate != "" && inits.InjectUserAmount > 0 {
			_ = req.SetInjectBatchUserTemplate(inits.InjectUserTemplate).InjectBatchDevSpace(
				inits.InjectUserAmount, inits.InjectUserAmountOffset,
			)
		}
		spinner.Stop()

		coloredoutput.Success("init demo data successfully \n")

		// wait for nocalhost-dep deployment in nocalhost-reserved namespace
		spinner = utils.NewSpinner(" waiting for Nocalhost-dep ready, this will take a few minutes...")
		spinner.Start()
		mustI(
			client.NameSpace(app.DefaultInitWaitNameSpace).WaitDeploymentToBeReady(app.DefaultInitWaitDeployment),
			"watch deployment timeout",
		)

		// change dep images tag
		setDepComponentDockerImage(kubectl, common.KubeConfig)

		spinner.Stop()

		coloredoutput.Success(
			"Nocalhost init completed. \n\n"+
				" Default user for plugin: \n"+
				" Api Server(Set on plugin): %s \n"+
				" Username: %s \n"+
				" Password: %s \n\n"+
				" Default administrator: \n"+
				" Web dashboard: %s\n"+
				" Username: %s \n"+
				" Password: %s \n\n"+
				" Now, you can setup VSCode plugin and enjoy Nocalhost! \n",
			req.BaseUrl,
			app.DefaultInitUserEmail,
			app.DefaultInitPassword,
			req.BaseUrl,
			app.DefaultInitAdminUserName,
			app.DefaultInitAdminPassWord,
		)

		must(req.IdleThePortForwardIfNeeded())
	},
}

func FindOutWebEndpoint(client *clientgoutils.ClientGoUtils) string {
	var port = inits.Port

	if inits.Type == "NodePort" {
		service, err := client.GetService("nocalhost-web")
		if err != nil || service == nil {
			log.Fatal("getting controller nocalhost-web from kubernetes failed")
			return ""
		}

		ports := service.Spec.Ports
		if len(ports) == 1 {
			port = int(ports[0].NodePort)
		}
	}

	// get Node ExternalIP
	nodes, err := client.GetNodesList()
	must(err)

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
	service, err := client.NameSpace(inits.NameSpace).GetService(app.DefaultInitNocalhostService)
	must(err)
	for _, ip := range service.Status.LoadBalancer.Ingress {
		if ip.IP != "" {
			loadBalancerIP = ip.IP
			break
		}
	}

	// should use loadbalancerIP > nodeExternalIP > nodeInternalIP
	endPoint := ""
	if nodeInternalIP != "" {
		endPoint = nodeInternalIP + ":" + strconv.Itoa(port)
	}
	if nodeExternalIP != "" {
		endPoint = nodeExternalIP + ":" + strconv.Itoa(port)
	}
	if loadBalancerIP != "" {
		endPoint = loadBalancerIP
		if inits.Port != 80 {
			endPoint = loadBalancerIP + ":" + strconv.Itoa(port)
		}
	}

	log.Debugf("Nocalhost get ready, endpoint is: %s \n", endPoint)

	return endPoint
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
		log.Debugf("Init nocalhost component with release %s", Version)
		*params = append(*params, "--set", "api.image.tag="+Version)
		*params = append(*params, "--set", "web.image.tag="+Version)
	} else {
		log.Debugf("Init nocalhost component with dev %s, but nocalhost-web with dev tag only", DevGitCommit)
		*params = append(*params, "--set", "api.image.tag="+DevGitCommit)
		// because of web image and api has different commitID, so take latest dev tag
		*params = append(*params, "--set", "web.image.tag=dev")
	}
}

// because of dep run latest docker image, so it can only use kubectl set
// image to set dep docker version same as nhctl version
func setDepComponentDockerImage(kubectl, kubeConfig string) {
	if Branch == "" {
		return
	}
	tag := Version
	// Branch will set by make nhctl
	if Branch != app.DefaultNocalhostMainBranch {
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
	_, err := tools.ExecCommand(nil, debug, false, false, kubectl, params...)
	utils.ShouldI(err, "set nocalhost-dep component tag fail")
}
