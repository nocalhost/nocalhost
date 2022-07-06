/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/pkg/nhctl/log"
)

//var container string

func init() {
	devContainersCmd.Flags().StringVarP(
		&common.WorkloadName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	devContainersCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	debugCmd.AddCommand(devContainersCmd)
}

var devContainersCmd = &cobra.Command{
	Use:   "containers [NAME]",
	Short: "Get containers of workload",
	Long:  `Get containers of workload`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		_, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(applicationName, common.WorkloadName, common.ServiceType)
		must(err)

		containerList, err := nocalhostSvc.GetOriginalContainers()
		must(err)
		var containers = make([]string, 0)
		for _, item := range containerList {
			containers = append(containers, item.Name)
		}
		if len(containers) == 0 {
			log.Fatal("Container num is 0??")
		}
		c, err := json.Marshal(containers)
		if err != nil {
			log.FatalE(err, "")
		}
		fmt.Println(string(c))
	},
}
