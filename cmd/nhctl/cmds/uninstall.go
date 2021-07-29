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
	"nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var force bool

func init() {
	uninstallCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	uninstallCmd.Flags().BoolVar(&force, "force", false, "force to uninstall anyway")
	rootCmd.AddCommand(uninstallCmd)
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall [NAME]",
	Short: "Uninstall application",
	Long:  `Uninstall application`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		if applicationName == _const.DefaultNocalhostApplication {
			log.Error(_const.DefaultNocalhostApplicationOperateErr)
			return
		}

		must(Prepare())

		appMeta, err := nocalhost.GetApplicationMeta(applicationName, nameSpace, kubeConfig)
		must(err)

		if appMeta.IsNotInstall() {
			log.Fatalf(appMeta.NotInstallTips())
			return
		}

		log.Info("Uninstalling application...")

		//goland:noinspection ALL
		mustI(appMeta.Uninstall(true), "Error while uninstall application")

		log.Infof("Application \"%s\" is uninstalled", applicationName)
	},
}
