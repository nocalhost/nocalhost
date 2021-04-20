/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	"os"
)

var (
	//settings     *app_flags.EnvSettings
	nameSpace    string
	debug        bool
	kubeConfig   string // the path to the kubeconfig file
	nocalhostApp *app.Application
)

func init() {

	//settings = app_flags.NewEnvSettings()
	rootCmd.PersistentFlags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", debug, "enable debug level log")
	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "the path of the kubeconfig file")

	//cobra.OnInitialize(func() {
	//})
}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl use to deploy coding project",
	Long:  `nhctl can deploy and develop application on Kubernetes. `,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		if debug {
			log.Init(zapcore.DebugLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		} else {
			log.Init(zapcore.InfoLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		}
		err := nocalhost.Init()
		if err != nil {
			log.FatalE(err, "Fail to init nhctl")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("hello nhctl")
	},
}

func Execute() {

	//str := "port-forward start bookinfo-coding -d ratings -p 12345:12345 --pod ratings-6848dcd688-wbn8l --way manual --kubeconfig ~/.nh/plugin/kubeConfigs/10_167_config"
	//str := "port-forward start coding-cd -d mariadb -p 3306:3306 --pod mariadb-0 --type statefulset --way manual --kubeconfig /Users/weiwang/.nh/plugin/kubeConfigs/7_73_config"
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
