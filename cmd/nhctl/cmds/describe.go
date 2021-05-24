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
	"gopkg.in/yaml.v3"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var deploy string

func init() {
	describeCmd.Flags().StringVarP(&deploy, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	describeCmd.Flags().StringVarP(&serviceType, "type", "t", "", "specify service type")
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
		initApp(applicationName)
		if deploy == "" {
			appProfile := nocalhostApp.GetDescription()
			bytes, err := yaml.Marshal(appProfile)
			if err == nil {
				fmt.Print(string(bytes))
			}
		} else {
			checkIfSvcExist(deploy, serviceType)
			appProfile := nocalhostSvc.GetDescription()
			bytes, err := yaml.Marshal(appProfile)
			if err == nil {
				fmt.Print(string(bytes))
			}
		}
	},
}
