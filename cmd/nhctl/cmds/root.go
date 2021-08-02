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
			_ = log.Init(zapcore.DebugLevel, nocalhost.GetLogDir(),
			_ = nocalhost.DefaultLogFileName)
		} else {
			_ = log.Init(zapcore.InfoLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		}
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

	if len(os.Args) == 1 {
		args := append([]string{"help"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
