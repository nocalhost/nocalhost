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

package app

import (
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/utils"
)

// Used by dep
type InstallEnvForDep struct {
	Global  []*profile.Env      `json:"global" yaml:"global"`
	Service []*ServiceEnvForDep `json:"service" yaml:"service"`
}

type ServiceEnvForDep struct {
	Name      string                `json:"name" yaml:"name"`
	Type      string                `json:"type" yaml:"type"`
	Container []*ContainerEnvForDep `json:"container" yaml:"container"`
}

type ContainerEnvForDep struct {
	Name       string         `json:"name" yaml:"name"`
	InstallEnv []*profile.Env `json:"installEnv" yaml:"installEnv"`
}

type ContainerDevEnv struct {
	DevEnv []*profile.Env
}

func (a *Application) GetDevContainerEnv(svcName, container string) *ContainerDevEnv {
	// Find service env
	devEnv := make([]*profile.Env, 0)
	kvMap := make(map[string]string, 0)
	serviceConfig, _ := a.GetSvcProfile(svcName)
	for _, v := range serviceConfig.ContainerConfigs {
		if v.Name == container || container == "" {
			if v.Dev.EnvFrom != nil && len(v.Dev.EnvFrom.EnvFile) > 0 {
				envFiles := make([]string, 0)
				for _, f := range v.Dev.EnvFrom.EnvFile {
					envFiles = append(envFiles, f.Path)
				}
				kvMap = utils.GetKVFromEnvFiles(envFiles)
			}
			// Env has a higher priority than envFrom
			for _, env := range v.Dev.Env {
				kvMap[env.Name] = env.Value
			}
		}
	}
	for k, v := range kvMap {
		env := &profile.Env{
			Name:  k,
			Value: v,
		}
		devEnv = append(devEnv, env)
	}
	return &ContainerDevEnv{DevEnv: devEnv}
}

func (a *Application) GetInstallEnvForDep() *InstallEnvForDep {
	appProfileV2, _ := a.GetProfile()

	envFiles := make([]string, 0)
	for _, f := range appProfileV2.EnvFrom.EnvFile {
		envFiles = append(envFiles, f.Path)
	}
	kvMap := utils.GetKVFromEnvFiles(envFiles)

	// Env has a higher priority than envFrom
	for _, env := range appProfileV2.Env {
		kvMap[env.Name] = env.Value
	}

	globalEnv := make([]*profile.Env, 0)
	for key, val := range kvMap {
		globalEnv = append(globalEnv, &profile.Env{
			Name:  key,
			Value: val,
		})
	}

	// Find service env
	servcesEnv := make([]*ServiceEnvForDep, 0)
	for _, svcProfile := range appProfileV2.SvcProfile {
		if svcProfile.ServiceConfigV2 == nil || len(svcProfile.ServiceConfigV2.ContainerConfigs) == 0 {
			continue
		}
		svcEnv := &ServiceEnvForDep{
			Name:      svcProfile.ActualName,
			Type:      string(svcProfile.Type),
			Container: make([]*ContainerEnvForDep, 0),
		}
		for _, config := range svcProfile.ServiceConfigV2.ContainerConfigs {
			if config.Install == nil {
				continue
			}
			if len(config.Install.Env) == 0 && len(config.Install.EnvFrom.EnvFile) == 0 {
				continue
			}

			envFiles1 := make([]string, 0)
			for _, f := range config.Install.EnvFrom.EnvFile {
				envFiles1 = append(envFiles1, f.Path)
			}
			kvMap1 := utils.GetKVFromEnvFiles(envFiles1)

			// Env has a higher priority than envFrom
			for _, env := range config.Install.Env {
				kvMap1[env.Name] = env.Value
			}

			containerEnv := make([]*profile.Env, 0)
			for key, val := range kvMap1 {
				containerEnv = append(containerEnv, &profile.Env{
					Name:  key,
					Value: val,
				})
			}

			svcEnv.Container = append(svcEnv.Container, &ContainerEnvForDep{
				Name:       config.Name,
				InstallEnv: containerEnv,
			})
		}
		if len(svcEnv.Container) > 0 {
			servcesEnv = append(servcesEnv, svcEnv)
		}
	}

	installEnv := &InstallEnvForDep{
		Global:  globalEnv,
		Service: servcesEnv,
	}
	return installEnv
}
