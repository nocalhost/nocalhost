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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/nocalhost"

	"nocalhost/pkg/nhctl/log"
)

func init() {
	devEndCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	debugCmd.AddCommand(devEndCmd)
}

var devEndCmd = &cobra.Command{
	Use:   "end [NAME]",
	Short: "end dev model",
	Long:  `end dev model`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, nil)

		meta, err := nocalhost.GetApplicationMetaInstalled(applicationName, nameSpace, kubeConfig)
		must(err)

		if b := meta.CheckIfDeploymentDeveloping(deployment); !b {
			log.Fatalf("Service %s is not in DevMode", deployment)
		}

		must(nocalhostApp.DevEnd(deployment, false))
		log.Infof("Service %s's DevMode has been ended", deployment)
	},
}
