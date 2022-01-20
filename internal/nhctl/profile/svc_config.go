/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package profile

type ServiceConfigV2 struct {
	Name                string               `validate:"required" json:"name" yaml:"name"`
	Type                string               `validate:"required" json:"serviceType" yaml:"serviceType"`
	PriorityClass       string               `json:"priorityClass,omitempty" yaml:"priorityClass,omitempty"`
	DependLabelSelector *DependLabelSelector `json:"dependLabelSelector,omitempty" yaml:"dependLabelSelector,omitempty"`
	ContainerConfigs    []*ContainerConfig   `validate:"dive" json:"containers" yaml:"containers"`
}

type ContainerConfig struct {
	Name    string                  `validate:"Container" json:"name" yaml:"name"`
	Hub     *HubConfig              `json:"hub" yaml:"hub,omitempty"`
	Install *ContainerInstallConfig `json:"install,omitempty" yaml:"install,omitempty"`
	Dev     *ContainerDevConfig     `json:"dev" yaml:"dev"`
}

func (s *ServiceConfigV2) GetContainerConfig(container string) *ContainerConfig {
	if s == nil {
		return nil
	}
	for _, c := range s.ContainerConfigs {
		if c.Name == container {
			return c
		}
	}
	return nil
}

func (s *ServiceConfigV2) GetDefaultContainerDevConfig() *ContainerDevConfig {
	if len(s.ContainerConfigs) == 0 {
		return nil
	}
	return s.ContainerConfigs[0].Dev
}

// GetContainerDevConfigOrDefault Compatible for v1
// Finding `containerName` config, if not found, use the first container config
func (s *ServiceConfigV2) GetContainerDevConfigOrDefault(containerName string) *ContainerDevConfig {
	if containerName == "" {
		return s.GetDefaultContainerDevConfig()
	}
	config := s.GetContainerDevConfig(containerName)
	if config == nil {
		config = s.GetDefaultContainerDevConfig()
	}
	return config
}

func (s *ServiceConfigV2) GetContainerDevConfig(containerName string) *ContainerDevConfig {
	for _, devConfig := range s.ContainerConfigs {
		if devConfig.Name == containerName {
			return devConfig.Dev
		}
	}
	return nil
}
