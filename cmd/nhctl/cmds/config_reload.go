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
	"nocalhost/pkg/nhctl/log"
)

func init() {
	configReloadCmd.Flags().StringVarP(
		&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	configReloadCmd.Flags().StringVarP(
		&serviceType, "controller-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	configCmd.AddCommand(configReloadCmd)
}

var configReloadCmd = &cobra.Command{
	Use:   "reload [Name]",
	Short: "reload application/service config",
	Long:  "reload application/service config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		initApp(commonFlags.AppName)

		if commonFlags.SvcName == "" {
			if err := nocalhostApp.ReloadCfg(true, false); err != nil {
				log.Fatal(errors.Wrap(err, ""))
			}
		} else {
			checkIfSvcExist(commonFlags.SvcName, serviceType)
			if err := nocalhostApp.ReloadSvcCfg(commonFlags.SvcName, nocalhostSvc.Type, true, false); err != nil {
				log.Fatal(errors.Wrap(err, ""))
			}
		}
	},
}
