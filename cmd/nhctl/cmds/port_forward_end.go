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
	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

var portForwardEndOptions = &app.PortForwardEndOptions{}

func init() {
	portForwardEndCmd.Flags().
		StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	portForwardEndCmd.Flags().
		StringVarP(&portForwardEndOptions.Port, "port", "p", "", "stop specify port-forward")
	portForwardEndCmd.Flags().
		StringVarP(&ServiceType, "type", "", "deployment", "specify service type")
	PortForwardCmd.AddCommand(portForwardEndCmd)
}

var portForwardEndCmd = &cobra.Command{
	Use:   "end [NAME]",
	Short: "stop port-forward",
	Long:  `stop port-forward`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initAppAndCheckIfSvcExist(applicationName, deployment, []string{ServiceType})
		err := nocalhostApp.StopPortForwardByPort(deployment, portForwardEndOptions.Port)
		if err != nil {
			log.Warnf("stop port-forward fail, %s", err.Error())
		}
		log.Infof("%s port-forward has been stop", portForwardEndOptions.Port)
	},
}
