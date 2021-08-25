/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/hub"
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

		originImage, err := nocalhostSvc.GetContainerImage(container)
		if err == nil {
			// load config from hub
			svcConfig, err := hub.FindNocalhostSvcConfig(nocalhostSvc.AppName, nocalhostSvc.Name, nocalhostSvc.Type, container, originImage)
			if err != nil {
				log.LogE(err)
			}
			if svcConfig != nil {
				if err := nocalhostSvc.UpdateSvcProfile(
					func(svcProfile *profile.SvcProfileV2) error {
						svcConfig.Name = nocalhostSvc.Name
						svcConfig.Type = string(nocalhostSvc.Type)
						svcProfile.ServiceConfigV2 = svcConfig
						return nil
					},
				); err != nil {
					log.Logf("Load nocalhost svc config from hub fail, fail while updating svc profile, err: %s", err.Error())
				}
			}
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
