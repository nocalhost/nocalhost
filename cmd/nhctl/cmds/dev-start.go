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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

var (
	nameSpace  string
	deployment string
)

var devStartOps = &app.DevStartOptions{}

func init() {

	devStartCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	devStartCmd.Flags().StringVarP(&devStartOps.DevLang, "lang", "l", "", "the development language, eg: java go python")
	devStartCmd.Flags().StringVarP(&devStartOps.DevImage, "image", "i", "", "image of development container")
	devStartCmd.Flags().StringVar(&devStartOps.WorkDir, "work-dir", "", "container's work dir")
	devStartCmd.Flags().StringVar(&devStartOps.SideCarImage, "sidecar-image", "", "image of sidecar container")
	debugCmd.AddCommand(devStartCmd)
}

var devStartCmd = &cobra.Command{
	Use:   "start [NAME]",
	Short: "enter dev model",
	Long:  `enter dev model`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if settings.Debug {
			log.SetLevel(logrus.DebugLevel)
		}
		applicationName := args[0]
		InitAppAndSvc(applicationName, deployment)

		if nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("\"%s\" is already developing", deployment)
		}

		nocalhostApp.CreateSvcProfile(deployment, app.Deployment)

		devStartOps.Kubeconfig = settings.KubeConfig
		fmt.Println("entering development model...")
		err = nocalhostApp.ReplaceImage(context.TODO(), deployment, devStartOps)
		if err != nil {
			log.Fatalf("fail to replace dev container: err%v\n", err)
		}
		err = nocalhostApp.SetDevelopingStatus(deployment, true)
		if err != nil {
			log.Fatal("fail to update \"developing\" status\n")
		}
	},
}
