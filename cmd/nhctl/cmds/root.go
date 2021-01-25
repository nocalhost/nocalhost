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
	"fmt"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/nocalhost"
	"os"

	"go.uber.org/zap/zapcore"

	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"

	"github.com/spf13/cobra"
)

var settings *app_flags.EnvSettings
var nocalhostApp *app.Application

func init() {

	settings = app_flags.NewEnvSettings()

	rootCmd.PersistentFlags().BoolVar(&settings.Debug, "debug", settings.Debug, "enable debug level log")
	rootCmd.PersistentFlags().StringVar(&settings.KubeConfig, "kubeconfig", "", "the path of the kubeconfig file")

	//cobra.OnInitialize(func() {
	//})
}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl use to deploy coding project",
	Long:  `nhctl can deploy and develop application on Kubernetes. `,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		err := nocalhost.Init()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fail to init: %s", err.Error())
			os.Exit(1)
		}
		if settings.Debug {
			log.Init(zapcore.DebugLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		} else {
			log.Init(zapcore.InfoLevel, nocalhost.GetLogDir(), nocalhost.DefaultLogFileName)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("hello nhctl")
	},
}

// Execute execute command
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
