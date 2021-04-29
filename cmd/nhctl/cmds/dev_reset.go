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
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	devResetCmd.Flags().StringVarP(&deployment, "deployment", "d", "",
		"k8s deployment which your developing service exists")
	devResetCmd.Flags().StringVarP(&serviceType, "svc-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet")
	debugCmd.AddCommand(devResetCmd)
}

var devResetCmd = &cobra.Command{
	Use:   "reset [NAME]",
	Short: "reset service",
	Long:  `reset service`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, serviceType)

		_ = nocalhostSvc.DevEnd(true)

		log.Infof("Service %s has been reset.\n", deployment)
	},
}
