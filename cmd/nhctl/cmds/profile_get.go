/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

func init() {
	profileGetCmd.Flags().StringVarP(&common.WorkloadName, "deployment", "d", "", "k8s workload name")
	profileGetCmd.Flags().StringVarP(&common.ServiceType, "type", "t", "deployment", "specify service type")
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
		nocalhostApp, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(args[0], common.WorkloadName, common.ServiceType)
		must(err)
		if configKey == "" {
			log.Fatal("--key must be specified")
		}
		if container == "" {
			log.Fatal("--container must be specified")
		}

		_ = nocalhostSvc.LoadConfigFromHubC(container)

		_ = nocalhostApp.ReloadSvcCfg(common.WorkloadName, nocalhostSvc.Type, false, true)

		switch configKey {
		case "image":
			p := nocalhostSvc.Config()

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
				var defaultIndex = -1
				for i, c := range p.ContainerConfigs {
					if c.Name == "" {
						defaultIndex = i
						break
					}
				}
				if defaultIndex >= 0 {
					p.ContainerConfigs[defaultIndex] = defaultContainerConfig
					defaultContainerConfig.Name = container // setting container name
				}
				must(nocalhostSvc.UpdateConfig(*p))
				fmt.Printf(`{"image": "%s"}`, defaultContainerConfig.Dev.Image)
			}
		default:
			log.Fatalf("Getting config key %s is unsupported", configKey)
		}
	},
}
