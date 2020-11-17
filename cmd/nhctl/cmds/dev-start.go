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
	"nocalhost/internal/nhctl"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os"
)

var (
	nameSpace  string
	deployment string
)

var devStartOps = &nhctl.DevStartOptions{}

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
		nocalhostApp, err = nhctl.NewApplication(applicationName)
		clientgoutils.Must(err)
		if deployment == "" {
			fmt.Println("error: please use -d to specify a k8s deployment")
			return
		}

		//if svcConfig != nil && svcConfig.DevImage != "" {
		//	devFlags.DevImage = svcConfig.DevImage
		//} else if devFlags.DevLang != "" {
		//	switch devFlags.DevLang {
		//	case "java":
		//		devFlags.DevImage = "roandocker/share-container-java:v3"
		//	case "ruby":
		//		devFlags.DevImage = "codingcorp-docker.pkg.coding.net/nocalhost/public/share-container-ruby:v1"
		//	default:
		//		fmt.Printf("unsupported language : %s\n", devFlags.DevLang)
		//		return
		//	}
		//} else {
		//	fmt.Println("[error] you mush specify a devImage by using -i flag or setting devImage in config or specifying a development language")
		//	return
		//}

		fmt.Println("entering development model...")
		err = nocalhostApp.ReplaceImage(deployment, devStartOps)
		if err != nil {
			fmt.Printf("[error] fail to replace dev container: err%v\n", err)
			os.Exit(1)
		}
	},
}
