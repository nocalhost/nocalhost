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

		switch configKey {
		case "image":
			p, err := nocalhostSvc.GetProfile()
			if err != nil {
				log.FatalE(err, "")
			}
			for _, c := range p.ContainerConfigs {
				if c.Name == container {
					if c.Dev != nil && c.Dev.Image != "" {
						fmt.Printf(`{"image": "%s"}`, c.Dev.Image)
					}
					return
				}
			}
		default:
			log.Fatalf("Getting config key %s is unsupported", configKey)
		}
	},
}
