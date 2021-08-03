/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/controller"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	"os"
	"strings"
)

var (
	nameSpace    string
	debug        bool
	kubeConfig   string // the path to the kubeconfig file
	nocalhostApp *app.Application
	nocalhostSvc *controller.Controller
)

func init() {

	rootCmd.PersistentFlags().StringVarP(
		&nameSpace, "namespace", "n", "",
		"kubernetes namespace",
	)
	rootCmd.PersistentFlags().BoolVar(
		&debug, "debug", debug,
		"enable debug level log",
	)
	rootCmd.PersistentFlags().StringVar(
		&kubeConfig, "kubeconfig", "",
		"the path of the kubeconfig file",
	)

}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl is a cloud-native development tool.",
	Long:  `nhctl is a cloud-native development tool.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if debug {
			_ = os.Setenv("_NOCALHOST_DEBUG_", "1")
			_ = log.Init(zapcore.DebugLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		} else {
			_ = log.Init(zapcore.InfoLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		}
		log.AddField("VERSION", Version)
		err := nocalhost.Init()
		if err != nil {
			log.FatalE(err, "Fail to init nhctl")
		}
		serviceType = strings.ToLower(serviceType)
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("hello nhctl")
	},
}

func Execute() {
	//str := "port-forward start bookinfo-coding -d ratings -p 12345:12345 --pod ratings-6848dcd688-wbn8l
	//--way manual --kubeconfig ~/.nh/plugin/kubeConfigs/10_167_config"
	//str := "port-forward start coding-cd -d mariadb -p 3306:3306 --pod mariadb-0 --type statefulset
	//--way manual --kubeconfig /Users/weiwang/.nh/plugin/kubeConfigs/7_73_config"
	//str := "init dep"
	//os.Args = append(os.Args, strings.Split(str, " ")...)
	if len(os.Args) == 1 {
		args := append([]string{"help"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
