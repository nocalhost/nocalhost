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
)

func init() {
	describeCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	describeCmd.Flags().StringVarP(&ServiceType, "type", "", "deployment", "specify service type")
	rootCmd.AddCommand(describeCmd)
}

var describeCmd = &cobra.Command{
	Use:   "describe [NAME]",
	Short: "Describe application info",
	Long:  `Describe application info`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		InitApp(applicationName)
		if deployment == "" {
			fmt.Print(nocalhostApp.GetDescription())
		} else {
			CheckIfSvcExist(deployment, ServiceType)
			fmt.Print(nocalhostApp.GetSvcDescription(deployment))
		}
	},
}
