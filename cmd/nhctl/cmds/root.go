/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/cmd/nhctl/cmds/install"
	"nocalhost/internal/nhctl/common/base"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/vpn/util"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	debug bool

	// pre check the nocalhost commands permissions
	authCheck bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&common.NameSpace, "namespace", "n", "",
		"kubernetes namespace",
	)
	rootCmd.PersistentFlags().BoolVar(
		&debug, "debug", debug,
		"enable debug level log",
	)
	rootCmd.PersistentFlags().BoolVar(
		&authCheck, "auth-check", authCheck,
		"pre check the nocalhost commands permissions, return yes"+
			" represent having enough permissions to call the command",
	)
	rootCmd.PersistentFlags().StringVar(
		&common.KubeConfig, "kubeconfig", "",
		"the path of the kubeconfig file",
	)

	rootCmd.AddCommand(install.UninstallCmd)

}

var cmdStartTime time.Time

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl is a cloud-native development tool.",
	Long:  `nhctl is a cloud-native development tool.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cmdStartTime = time.Now()

		// Init log
		if debug {
			_ = os.Setenv("_NOCALHOST_DEBUG_", "1")
			_ = log.Init(zapcore.DebugLevel, nocalhost.GetLogDir(), _const.DefaultLogFileName)
		} else {
			_ = log.Init(zapcore.InfoLevel, nocalhost.GetLogDir(), _const.DefaultLogFileName)
		}
		log.AddField("VERSION", Version)
		log.AddField("COMMIT", GitCommit)
		log.AddField("BRANCH", Branch)
		log.AddField("ARGS", strings.Join(os.Args, " "))

		var esUrl string
		bys, err := ioutil.ReadFile(filepath.Join(nocalhost_path.GetNhctlHomeDir(), "config"))
		if err == nil && len(bys) > 0 {
			configFile := base.ConfigFile{}
			err = yaml.Unmarshal(bys, &configFile)
			if err == nil && configFile.NhEsUrl != "" {
				esUrl = configFile.NhEsUrl
			}
		}
		if esUrl == "" {
			esUrl = os.Getenv("NH_ES_URL")
		}
		if esUrl != "" {
			log.InitEs(esUrl)
		}
		if !util.IsAdmin() {
			err = nocalhost.Init()
			if err != nil {
				log.FatalE(err, "Fail to init nhctl")
			}
		}
		common.ServiceType = strings.ToLower(common.ServiceType)

		if authCheck {
			must(common.Prepare())
			if clientgoutils.AuthCheck(common.NameSpace, common.KubeConfig, cmd) {
				fmt.Printf("yes")
			}
			os.Exit(0)
			return
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if os.Getenv("_NOCALHOST_DEBUG_") != "" || os.Getenv("NH_ES_URL") != "" {
			d := time.Now().Sub(cmdStartTime)
			cmds := clientgoutils.GetCmd(cmd, nil)

			cmd.Flags().Visit(
				func(flag *pflag.Flag) {
					cmds = append(cmds, "-"+flag.Name)
					cmds = append(cmds, flag.Value.String())
				},
			)

			field := make(map[string]interface{}, 0)
			field["cost"] = d.Milliseconds()
			log.WriteToEsWithField(field, "[TimeMachine] %v, cost: %dms", cmds, d.Milliseconds())
		}
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
