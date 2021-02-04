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
	"gopkg.in/yaml.v2"

	"nocalhost/internal/nhctl/app"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	configGetCmd.Flags().StringVarP(&commonFlags.SvcName, "deployment", "d", "", "k8s deployment which your developing service exists")
	configCmd.AddCommand(configGetCmd)
}

type ConfigForPlugin struct {
	Services []*app.ServiceConfigV2 `json:"services" yaml:"services"`
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
		InitApp(commonFlags.AppName)

		if commonFlags.SvcName == "" {
			config := &ConfigForPlugin{}
			config.Services = make([]*app.ServiceConfigV2, 0)
			for _, svcPro := range nocalhostApp.AppProfileV2.SvcProfile {
				config.Services = append(config.Services, svcPro.ServiceConfigV2)
			}
			bys, err := yaml.Marshal(config)
			if err != nil {
				log.FatalE(errors.Wrap(err, ""), "fail to get application config")
			}
			fmt.Println(string(bys))

		} else {
			CheckIfSvcExist(commonFlags.SvcName)
			svcProfile := nocalhostApp.GetSvcProfileV2(commonFlags.SvcName)
			if svcProfile != nil {
				bys, err := yaml.Marshal(svcProfile.ServiceConfigV2)
				if err != nil {
					log.FatalE(errors.Wrap(err, ""), "fail to get svc profile")
				}
				fmt.Println(string(bys))
			}
		}
	},
}
