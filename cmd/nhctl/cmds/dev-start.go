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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
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
		applicationName := args[0]
		nocalhostApp, err = app.NewApplication(applicationName)
		clientgoutils.Must(err)
		if deployment == "" {
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}

		exist, err := nocalhostApp.CheckIfSvcExist(deployment, app.Deployment)
		if err != nil {
			printlnErr("fail to check if svc exist", err)
			os.Exit(1)
		} else if !exist {
			fmt.Printf("\"%s\" not found\n", deployment)
			os.Exit(1)
		}

		nocalhostApp.CreateSvcProfile(deployment, app.Deployment)

		fmt.Println("entering development model...")
		err = nocalhostApp.ReplaceImage(deployment, devStartOps)
		if err != nil {
			fmt.Printf("[error] fail to replace dev container: err%v\n", err)
			os.Exit(1)
		}
		err = nocalhostApp.SetDevelopingStatus(deployment, true)
		if err != nil {
			fmt.Printf("[error] fail to update \"developing\" status\n")
			os.Exit(1)
		}
	},
}
