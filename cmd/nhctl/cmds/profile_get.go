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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	profileGetCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s workload name")
	profileGetCmd.Flags().StringVarP(&serviceType, "type", "t", "deployment", "specify service type")
	profileGetCmd.Flags().StringVarP(&container, "container", "c", "", "container name of pod")
	profileGetCmd.Flags().StringVarP(&configKey, "key", "k", "", "key of dev config")
	profileCmd.AddCommand(profileGetCmd)
}

var profileGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a config item of Profile",
	Long:  `Get a config item of Profile`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		initAppAndCheckIfSvcExist(args[0], deployment, serviceType)
		if configKey == "" {
			log.Fatal("--key must be specified")
		}
		if container == "" {
			log.Fatal("--container must be specified")
		}

		_ = nocalhostApp.ReloadSvcCfg(deployment, nocalhostSvc.Type, false, true)

		switch configKey {
		case "image":
			p, err := nocalhostSvc.GetProfile()
			if err != nil {
				log.FatalE(err, "")
			}
			var defaultContainerConfig *profile.ContainerConfig
			for _, c := range p.ContainerConfigs {
				if c.Name == container {
					if c.Dev != nil && c.Dev.Image != "" {
						fmt.Printf(`{"image": "%s"}`, c.Dev.Image)
					}
					return
				} else if c.Name == "" {
					defaultContainerConfig = c
				}
			}
			if defaultContainerConfig != nil && defaultContainerConfig.Dev != nil && defaultContainerConfig.Dev.Image != "" {
				must(
					nocalhostSvc.UpdateSvcProfile(
						func(v2 *profile.SvcProfileV2) error {
							var defaultIndex = -1
							for i, c := range v2.ContainerConfigs {
								if c.Name == "" {
									defaultIndex = i
								}
							}
							if defaultIndex >= 0 {
								v2.ContainerConfigs[defaultIndex] = defaultContainerConfig
								defaultContainerConfig.Name = container // setting container name
								return nil
							}
							return nil
						},
					),
				)
				fmt.Printf(`{"image": "%s"}`, defaultContainerConfig.Dev.Image)
			}
		default:
			log.Fatalf("Getting config key %s is unsupported", configKey)
		}
	},
}
