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
	"nocalhost/pkg/nhctl/utils"
)

// Used by dep
type InstallEnvForDep struct {
	Global  []*Env              `json:"global" yaml:"global"`
	Service []*ServiceEnvForDep `json:"service" yaml:"service"`
}

type ServiceEnvForDep struct {
	Name      string                `json:"name" yaml:"name"`
	Type      string                `json:"type" yaml:"type"`
	Container []*ContainerEnvForDep `json:"container" yaml:"container"`
}

type ContainerEnvForDep struct {
	Name       string `json:"name" yaml:"name"`
	InstallEnv []*Env `json:"installEnv" yaml:"installEnv"`
}

type ContainerDevEnv struct {
	DevEnv []*Env
}

func (a *Application) GetDevContainerEnv(svcName, container string) *ContainerDevEnv {
	// Find service env
	devEnv := make([]*Env, 0)
	kvMap := make(map[string]string, 0)
	serviceConfig := a.GetSvcProfileV2(svcName)
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
	if len(kvMap) > 0 {
		for k, v := range kvMap {
			env := &Env{
				Name:  k,
				Value: v,
			}
			devEnv = append(devEnv, env)
		}
	}
	return &ContainerDevEnv{DevEnv: devEnv}
}

func (a *Application) GetInstallEnvForDep() *InstallEnvForDep {

	envFiles := make([]string, 0)
	for _, f := range a.AppProfileV2.EnvFrom.EnvFile {
		envFiles = append(envFiles, f.Path)
	}
	kvMap := utils.GetKVFromEnvFiles(envFiles)

	// Env has a higher priority than envFrom
	for _, env := range a.AppProfileV2.Env {
		kvMap[env.Name] = env.Value
	}

	globalEnv := make([]*Env, 0)
	for key, val := range kvMap {
		globalEnv = append(globalEnv, &Env{
			Name:  key,
			Value: val,
		})
	}

	// Find service env
	servcesEnv := make([]*ServiceEnvForDep, 0)
	for _, svcProfile := range a.AppProfileV2.SvcProfile {
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

			containerEnv := make([]*Env, 0)
			for key, val := range kvMap1 {
				containerEnv = append(containerEnv, &Env{
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
