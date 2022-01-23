/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"

	"nocalhost/cmd/nhctl/cmds/tpl"
)

type CommonFlags struct {
	AppName   string
	SvcName   string
	AppConfig bool
}

var commonFlags = CommonFlags{}

func init() {
	configTemplateCmd.Flags().StringVarP(&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists")
	configTemplateCmd.Flags().StringVarP(&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet")
	configCmd.AddCommand(configTemplateCmd)
}

var configTemplateCmd = &cobra.Command{
	Use:   "template [Name]",
	Short: "get service config template",
	Long:  "get service config template",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		common.InitAppAndCheckIfSvcExist(commonFlags.AppName, commonFlags.SvcName, common.ServiceType)
		t, err := tpl.GetSvcTpl(commonFlags.SvcName)
		mustI(err, "fail to get controller tpl")
		fmt.Println(t)
	},
}
