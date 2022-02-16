/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

var configKey string
var configVal string

const (
	imageKey  = "image"
	gitUrlKey = "gitUrl"
)

func init() {
	profileSetCmd.Flags().StringVarP(&common.WorkloadName, "deployment", "d", "", "k8s workload name")
	profileSetCmd.Flags().StringVarP(&common.ServiceType, "type", "t", "deployment", "specify service type")
	profileSetCmd.Flags().StringVarP(&container, "container", "c", "", "container name of pod")
	profileSetCmd.Flags().StringVarP(&configKey, "key", "k", "", "key of dev config")
	profileSetCmd.Flags().StringVarP(&configVal, "value", "v", "", "value of dev config")
	profileCmd.AddCommand(profileSetCmd)
}

var profileSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a config item of Profile",
	Long:  `Set a config item of Profile`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		_, nocalhostSvc, err := common.InitAppAndCheckIfSvcExist(args[0], common.WorkloadName, common.ServiceType)
		must(err)
		if configKey == "" {
			log.Fatal("--key must be specified")
		}
		if configVal == "" {
			log.Fatal("--value must be specified")
		}
		if container == "" {
			log.Fatal("--container must be specified")
		}

		supportedConfigKey := []string{imageKey, gitUrlKey}
		if !stringSliceContains(supportedConfigKey, configKey) {
			log.Fatalf("Config key %s is unsupported", configKey)
		}

		svcConfig := nocalhostSvc.Config()
		var defaultContainerConfig, targetContainerConfig *profile.ContainerConfig
		for _, c := range svcConfig.ContainerConfigs {
			if c.Name == "" {
				defaultContainerConfig = c
			} else if c.Name == container {
				targetContainerConfig = c
				break
			}
		}
		if targetContainerConfig == nil && defaultContainerConfig != nil {
			defaultContainerConfig.Name = container
			targetContainerConfig = defaultContainerConfig
		}
		if targetContainerConfig != nil {
			if targetContainerConfig.Dev == nil {
				targetContainerConfig.Dev = &profile.ContainerDevConfig{}
			}
			if configKey == imageKey {
				targetContainerConfig.Dev.Image = configVal
			} else if configKey == gitUrlKey {
				targetContainerConfig.Dev.GitUrl = configVal
			}
			must(nocalhostSvc.UpdateConfig(*svcConfig))
			return
		}
		targetContainerConfig = &profile.ContainerConfig{Dev: &profile.ContainerDevConfig{}, Name: container}
		switch configKey {
		case imageKey:
			targetContainerConfig.Dev.Image = configVal
		case gitUrlKey:
			targetContainerConfig.Dev.GitUrl = configVal
		}
		svcConfig.ContainerConfigs = append(svcConfig.ContainerConfigs, targetContainerConfig)
		must(nocalhostSvc.UpdateConfig(*svcConfig))
	},
}

func stringSliceContains(ss []string, item string) bool {
	for _, s := range ss {
		if s == item {
			return true
		}
	}
	return false
}
