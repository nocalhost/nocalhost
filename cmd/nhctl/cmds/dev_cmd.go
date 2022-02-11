/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/pkg/nhctl/log"
)

type DevCommandType string

const (
	buildCommand          DevCommandType = "build"
	runCommand            DevCommandType = "run"
	debugCommand          DevCommandType = "debug"
	hotReloadRunCommand   DevCommandType = "hotReloadRun"
	hotReloadDebugCommand DevCommandType = "hotReloadDebug"
)

var commandType string
var container string

func init() {
	devCmdCmd.Flags().StringVarP(&common.WorkloadName, "deployment", "d", "",
		"K8s deployment which your developing service exists")
	devCmdCmd.Flags().StringVarP(&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
	devCmdCmd.Flags().StringVarP(&container, "container", "c", "",
		"which container of pod to run command")
	devCmdCmd.Flags().StringVar(&commandType, "dev-command-type", "", fmt.Sprintf(
		"Dev command type can be: %s, %s, %s, %s, %s",
		buildCommand, runCommand, debugCommand, hotReloadRunCommand, hotReloadDebugCommand))
	debugCmd.AddCommand(devCmdCmd)
}

var devCmdCmd = &cobra.Command{
	Use:   "cmd [NAME]",
	Short: "Run cmd in dev container",
	Long:  `Run cmd in dev container`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if commandType == "" {
			log.Fatal("--dev-command-type mush be specified")
		}
		applicationName := args[0]
		nocalhostApp, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)
		must(err)

		if !nocalhostSvc.IsInDevMode() {
			log.Fatalf("%s is not in DevMode", common.WorkloadName)
		}

		svcConfig := nocalhostSvc.Config()

		if svcConfig.GetContainerDevConfigOrDefault(container) == nil ||
			svcConfig.GetContainerDevConfigOrDefault(container).Command == nil {
			log.Fatalf("%s command not defined", commandType)
		}
		var targetCommand []string
		switch commandType {
		case string(buildCommand):
			targetCommand = svcConfig.GetContainerDevConfigOrDefault(container).Command.Build
		case string(runCommand):
			targetCommand = svcConfig.GetContainerDevConfigOrDefault(container).Command.Run
		case string(debugCommand):
			targetCommand = svcConfig.GetContainerDevConfigOrDefault(container).Command.Debug
		case string(hotReloadDebugCommand):
			targetCommand = svcConfig.GetContainerDevConfigOrDefault(container).Command.HotReloadDebug
		case string(hotReloadRunCommand):
			targetCommand = svcConfig.GetContainerDevConfigOrDefault(container).Command.HotReloadRun
		default:
			log.Fatalf("%s is not supported", commandType)

		}
		if len(targetCommand) == 0 {
			log.Fatalf("%s command not defined", commandType)
		}
		podList, err := nocalhostSvc.GetPodList()
		if err != nil {
			log.Fatal(err)
		}
		runningPod := make([]v1.Pod, 0, 1)
		for _, item := range podList {
			if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
				runningPod = append(runningPod, item)
			}
		}
		if len(runningPod) != 1 {
			log.Fatalf("pod number: %d, is not 1, please make sure pod number is 1", len(runningPod))
		}
		must(nocalhostApp.Exec(runningPod[0], container, targetCommand))
	},
}
