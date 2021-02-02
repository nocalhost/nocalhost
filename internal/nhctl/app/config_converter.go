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
)

func ConvertConfigFileV1ToV2(srcFile string, destFile string) error {
	bytes, err := ioutil.ReadFile(srcFile)
	if err != nil {
		return errors.Wrap(err, "")
	}

	config := &NocalHostAppConfig{}
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

func convertConfigV1ToV2(configV1 *NocalHostAppConfig) (*NocalHostAppConfigV2, error) {
	if configV1 == nil {
		return nil, errors.New("V1 config can not be nil")
	}

	configV2 := &NocalHostAppConfigV2{
		ConfigProperties: &ConfigProperties{
			Version: "v2",
			EnvFile: "",
		},
		ApplicationConfig: &ApplicationConfig{
			Name:           configV1.Name,
			Type:           configV1.Type,
			ResourcePath:   configV1.ResourcePath,
			IgnoredPath:    configV1.IgnoredPath,
			PreInstall:     configV1.PreInstall,
			HelmValues:     nil,
			Env:            nil,
			EnvFrom:        nil,
			ServiceConfigs: nil,
		},
	}

	serviceConfigs := make([]*ServiceConfigV2, 0)
	for _, svcV1 := range configV1.SvcConfigs {
		svcV2 := &ServiceConfigV2{
			Name:                svcV1.Name,
			Type:                svcV1.Type,
			PriorityClass:       svcV1.PriorityClass,
			DependLabelSelector: nil,
			ContainerConfigs: []*ContainerConfig{
				{
					Name:    "",
					Install: nil,
					Dev: &ContainerDevConfig{
						GitUrl:                svcV1.GitUrl,
						Image:                 svcV1.DevImage,
						Shell:                 svcV1.DevContainerShell,
						WorkDir:               svcV1.WorkDir,
						DevContainerResources: svcV1.DevContainerResources,
						PersistentVolumeDirs:  svcV1.PersistentVolumeDirs,
						Command: &DevCommands{
							Build:          svcV1.BuildCommand,
							Run:            svcV1.RunCommand,
							Debug:          svcV1.DebugCommand,
							HotReloadRun:   svcV1.HotReloadRunCommand,
							HotReloadDebug: svcV1.HotReloadDebugCommand,
						},
						DebugConfig:     nil,
						UseDevContainer: false,
						Sync: &SyncConfig{
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
			svcV2.DependLabelSelector = &DependLabelSelector{
				Pods: svcV1.Pods,
				Jobs: svcV1.Jobs,
			}
		}

		serviceConfigs = append(serviceConfigs, svcV2)
	}

	configV2.ApplicationConfig.ServiceConfigs = serviceConfigs

	return configV2, nil
}
