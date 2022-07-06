/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/dev_dir"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

var notificationPrefix = `# This is the runtime configuration which stored in K8s cluster. Modifications 
# to the development configuration will take effect the next time you enter
# the DevMode, and modification will share with all those who use this cluster.
#
# If you want to customized personal configuration, you can create a configuration
# file named config.yaml in the root directory of your project under the
# folder .nocalhost (/.nocalhost/config.yaml). It will become part of your 
# project, you can easily share configuration with other developers, or
# develop on any other devices using this personal configuration.
#`

var svcNotificationTips = `
# Tips: You can paste the configuration follow into 
# %s
#`

var notificationSuffix = `
# In addition, if you want to config multi service in same config.yaml, or use
# the Server-version of Nocalhost, you can also configure under the definition 
# of the application, such as:
# https://nocalhost.dev/docs/config/config-deployment-quickstart
#`

var svcNotificationTipsLocalLoaded = `# Tips: This configuration is a in-memory replica of local file: 
# 
# '%s'
# 
# You should modify your configuration in local file, and the modification will
# take effect immediately. (Dev modification will take effect the next time you enter the DevMode)
#`

var svcNotificationTipsCmLoaded = `# Tips: This configuration is a in-memory replica of configmap: 
# 
# '%s'
# 
# You should modify your configuration in configmap, and the modification will
# take effect immediately. (Dev modification will take effect the next time you enter the DevMode)
#`

var svcNotificationTipsAnnotationLoaded = `# Tips: This configuration is a in-memory replica of annotation: 
# 
# annotations:
#   %s: |
#     [Your Config]
# 
# You should modify your configuration in resource's annotation', and the modification will
# take effect immediately. (Dev modification will take effect the next time you enter the DevMode)
#`

func init() {
	configGetCmd.Flags().StringVarP(
		&commonFlags.SvcName, "deployment", "d", "",
		"k8s deployment which your developing service exists",
	)
	configGetCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	configGetCmd.Flags().BoolVar(
		&commonFlags.AppConfig, "app-config", false,
		"get application config",
	)
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
		nocalhostApp, err := common.InitAppMutate(commonFlags.AppName)
		if err != nil {
			log.Logf("init app:%s on namespace: %s, error: %v", commonFlags.AppName, common.NameSpace, err)
			return
		}
		// get application config
		if commonFlags.AppConfig {

			applicationConfig := nocalhostApp.GetApplicationConfigV2()
			bys, err := yaml.Marshal(applicationConfig)
			must(errors.Wrap(err, "fail to get application config"))
			fmt.Println(string(bys))
			return
		}

		if commonFlags.SvcName == "" {
			appConfig := nocalhostApp.GetApplicationConfigV2()
			config := &ConfigForPlugin{}
			config.Services = make([]*profile.ServiceConfigV2, 0)
			for _, svcPro := range appConfig.ServiceConfigs {
				config.Services = append(config.Services, svcPro)
			}
			bys, err := yaml.Marshal(config)
			must(errors.Wrap(err, "fail to get application config"))
			fmt.Println(string(bys))

		} else {
			nocalhostSvc, err := nocalhostApp.InitAndCheckIfSvcExist(commonFlags.SvcName, common.ServiceType)
			must(err)

			_ = nocalhostSvc.LoadConfigFromHub()
			// need to load latest config
			_ = nocalhostApp.ReloadSvcCfg(commonFlags.SvcName, nocalhostSvc.Type, false, true)
			nocalhostSvc.ReloadConfig()

			svcProfile, err := nocalhostSvc.GetProfile()
			must(err)
			svcConfig := nocalhostSvc.Config()

			// to avoid empty config
			if svcConfig == nil {
				svcConfig = &profile.ServiceConfigV2{
					Name:             commonFlags.SvcName,
					Type:             nocalhostSvc.Type.String(),
					ContainerConfigs: []*profile.ContainerConfig{},
				}
			}

			if svcProfile != nil {
				bys, err := yaml.Marshal(svcConfig)
				must(errors.Wrap(err, "fail to marshal svc config"))

				pack := dev_dir.NewSvcPack(
					nocalhostSvc.NameSpace,
					nocalhostSvc.AppMeta.NamespaceId,
					nocalhostSvc.AppName,
					nocalhostSvc.Type,
					nocalhostSvc.Name,
					"",
				)

				path := fp.NewFilePath(string(pack.GetAssociatePath())).
					RelOrAbs(".nocalhost").
					RelOrAbs("config.yaml").Path

				notification := ""
				if svcProfile.LocalConfigLoaded {
					notification += fmt.Sprintf(
						svcNotificationTipsLocalLoaded,
						path,
					)
					notification += notificationSuffix
				} else if svcProfile.CmConfigLoaded {
					notification += fmt.Sprintf(
						svcNotificationTipsCmLoaded,
						appmeta.ConfigMapName(commonFlags.AppName),
					)
				} else if svcProfile.AnnotationsConfigLoaded {
					notification += fmt.Sprintf(
						svcNotificationTipsAnnotationLoaded,
						appmeta.AnnotationKey,
					)
				} else {
					notification += notificationPrefix
					notification += fmt.Sprintf(
						svcNotificationTips,
						path,
					)
					notification += notificationSuffix
				}

				fmt.Println(
					fmt.Sprintf(
						"%s \n%s", notification, string(bys),
					),
				)
			}
		}
	},
}
