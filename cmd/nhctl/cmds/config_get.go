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
	"nocalhost/internal/nhctl/profile"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	configGetCmd.Flags().StringVarP(&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists")
	configGetCmd.Flags().StringVarP(&serviceType, "svc-type", "t", "",
		"kind of k8s controller,such as deployment,statefulSet")
	configGetCmd.Flags().BoolVar(&commonFlags.AppConfig, "app-config", false,
		"get application config")
	configCmd.AddCommand(configGetCmd)
}

type ConfigForPlugin struct {
	Services []*profile.ServiceConfigV2 `json:"services" yaml:"services"`
}

var configGetCmd = &cobra.Command{
	Use:   "get [Name]",
	Short: "get application/service config",
	Long:  "get application/service config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		initApp(commonFlags.AppName)

		// get application config
		if commonFlags.AppConfig {
			applicationConfig := nocalhostApp.GetAppProfileV2()
			bys, err := yaml.Marshal(applicationConfig)
			must(errors.Wrap(err, "fail to get application config"))
			fmt.Println(string(bys))
			return
		}

		appProfile, err := nocalhostApp.GetProfile()
		must(err)
		if commonFlags.SvcName == "" {
			config := &ConfigForPlugin{}
			config.Services = make([]*profile.ServiceConfigV2, 0)
			for _, svcPro := range appProfile.SvcProfile {
				config.Services = append(config.Services, svcPro.ServiceConfigV2)
			}
			bys, err := yaml.Marshal(config)
			must(errors.Wrap(err, "fail to get application config"))
			fmt.Println(string(bys))

		} else {
			checkIfSvcExist(commonFlags.SvcName, serviceType)
			svcProfile := appProfile.SvcProfileV2(commonFlags.SvcName, serviceType)
			if svcProfile != nil {
				bys, err := yaml.Marshal(svcProfile.ServiceConfigV2)
				must(errors.Wrap(err, "fail to get svc profile"))
				fmt.Println(string(bys))
			}
		}
	},
}
