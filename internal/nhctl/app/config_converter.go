/*
Copyright 2021 The Nocalhost Authors.
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

package app

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	profile2 "nocalhost/internal/nhctl/profile"
)

func ConvertAppProfileFileV1ToV2(srcFile string, destFile string) error {
	bytes, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return errors.Wrap(err, "")
	}

	profile := &profile2.AppProfile{}
	err = yaml.Unmarshal(bytes, profile)
	if err != nil {
		return errors.Wrap(err, "")
	}

	profileV2, err := convertProfileV1ToV2(profile)
	if err != nil {
		return err
	}

	v2Bytes, err := yaml.Marshal(profileV2)
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = ioutil.WriteFile(destFile, v2Bytes, 0644)
	return errors.Wrap(err, "")
}

func ConvertConfigFileV1ToV2(srcFile string, destFile string) error {
	bytes, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return errors.Wrap(err, "")
	}

	config := &profile2.NocalHostAppConfig{}
	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return errors.Wrap(err, "")
	}

	configV2, err := convertConfigV1ToV2(config)
	if err != nil {
		return err
	}

	v2Bytes, err := yaml.Marshal(configV2)
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = ioutil.WriteFile(destFile, v2Bytes, 0644)
	return errors.Wrap(err, "")
}

func convertProfileV1ToV2(profileV1 *profile2.AppProfile) (*profile2.AppProfileV2, error) {
	if profileV1 == nil {
		return nil, errors.New("V1 profile can not be nil")
	}

	profileV2 := &profile2.AppProfileV2{
		Name:                    profileV1.Name,
		ChartName:               profileV1.ChartName,
		ReleaseName:             profileV1.ReleaseName,
		Namespace:               profileV1.Namespace,
		Kubeconfig:              profileV1.Kubeconfig,
		DependencyConfigMapName: profileV1.DependencyConfigMapName,
		SvcProfile:              nil,
		Installed:               profileV1.Installed,
		ResourcePath:            profileV1.ResourcePath,
		IgnoredPath:             profileV1.IgnoredPath,
	}

	svcProfiles := make([]*profile2.SvcProfileV2, 0)
	for _, svcProfileV1 := range profileV1.SvcProfile {
		svcProfileV2 := &profile2.SvcProfileV2{
			ServiceConfigV2:                        convertServiceConfigV1ToV2(svcProfileV1.ServiceDevOptions),
			ContainerProfile:                       nil,
			ActualName:                             svcProfileV1.ActualName,
			Developing:                             svcProfileV1.Developing,
			PortForwarded:                          svcProfileV1.PortForwarded,
			Syncing:                                svcProfileV1.Syncing,
			RemoteSyncthingPort:                    svcProfileV1.RemoteSyncthingPort,
			RemoteSyncthingGUIPort:                 svcProfileV1.RemoteSyncthingGUIPort,
			SyncthingSecret:                        svcProfileV1.SyncthingSecret,
			LocalSyncthingPort:                     svcProfileV1.LocalSyncthingPort,
			LocalSyncthingGUIPort:                  svcProfileV1.LocalSyncthingGUIPort,
			LocalAbsoluteSyncDirFromDevStartPlugin: svcProfileV1.LocalAbsoluteSyncDirFromDevStartPlugin,
			//DevPortList:                            svcProfileV1.DevPortList,
			//PortForwardStatusList: svcProfileV1.PortForwardStatusList,
			//PortForwardPidList: svcProfileV1.PortForwardPidList,
		}
		svcProfiles = append(svcProfiles, svcProfileV2)
	}

	profileV2.SvcProfile = svcProfiles
	return profileV2, nil
}
func convertConfigV1ToV2(configV1 *profile2.NocalHostAppConfig) (*profile2.NocalHostAppConfigV2, error) {
	if configV1 == nil {
		return nil, errors.New("V1 config can not be nil")
	}

	configV2 := &profile2.NocalHostAppConfigV2{
		ConfigProperties: &profile2.ConfigProperties{
			Version: "v2",
			EnvFile: "",
		},
		ApplicationConfig: &profile2.ApplicationConfig{
			Name:           configV1.Name,
			Type:           configV1.Type,
			ResourcePath:   configV1.ResourcePath,
			IgnoredPath:    configV1.IgnoredPath,
			PreInstall:     configV1.PreInstall,
			HelmValues:     nil,
			Env:            nil,
			ServiceConfigs: nil,
		},
	}

	serviceConfigs := make([]*profile2.ServiceConfigV2, 0)
	for _, svcV1 := range configV1.SvcConfigs {
		svcV2 := convertServiceConfigV1ToV2(svcV1)
		serviceConfigs = append(serviceConfigs, svcV2)
	}

	configV2.ApplicationConfig.ServiceConfigs = serviceConfigs

	return configV2, nil
}

func convertServiceConfigV1ToV2(svcV1 *profile2.ServiceDevOptions) *profile2.ServiceConfigV2 {
	svcV2 := &profile2.ServiceConfigV2{
		Name:                svcV1.Name,
		Type:                svcV1.Type,
		PriorityClass:       svcV1.PriorityClass,
		DependLabelSelector: nil,
		ContainerConfigs: []*profile2.ContainerConfig{
			{
				Name:    "",
				Install: nil,
				Dev: &profile2.ContainerDevConfig{
					GitUrl:                svcV1.GitUrl,
					Image:                 svcV1.DevImage,
					Shell:                 svcV1.DevContainerShell,
					WorkDir:               svcV1.WorkDir,
					DevContainerResources: svcV1.DevContainerResources,
					PersistentVolumeDirs:  svcV1.PersistentVolumeDirs,
					Command: &profile2.DevCommands{
						Build:          svcV1.BuildCommand,
						Run:            svcV1.RunCommand,
						Debug:          svcV1.DebugCommand,
						HotReloadRun:   svcV1.HotReloadRunCommand,
						HotReloadDebug: svcV1.HotReloadDebugCommand,
					},
					DebugConfig:     nil,
					UseDevContainer: false,
					Sync: &profile2.SyncConfig{
						Type:              "",
						FilePattern:       svcV1.SyncedPattern,
						IgnoreFilePattern: svcV1.IgnoredPattern,
					},
					Env:         nil,
					EnvFrom:     nil,
					PortForward: svcV1.DevPort,
				},
			},
		},
	}
	if len(svcV1.Jobs) > 0 || len(svcV1.Pods) > 0 {
		svcV2.DependLabelSelector = &profile2.DependLabelSelector{
			Pods: svcV1.Pods,
			Jobs: svcV1.Jobs,
		}
	}
	return svcV2
}

func checkConfigVersion(content string) (string, error) {
	config := &profile2.NocalHostAppConfigV2{}

	// ignored err to prevent un strict yaml
	_ = yaml.Unmarshal([]byte(content), config)

	if config.ConfigProperties != nil {
		return config.ConfigProperties.Version, nil
	}
	return "v1", nil
}
