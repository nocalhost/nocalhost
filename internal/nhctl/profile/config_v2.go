/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package profile

import (
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/pkg/nhctl/clientgoutils"
	"strings"
)

//type AppType string
//type SvcType string

var (
	gitIgnoreMode = "gitIgnore"
	PatternMode   = "pattern"
)

type NocalHostAppConfigV2 struct {
	ConfigProperties  ConfigProperties  `json:"configProperties" yaml:"configProperties"`
	Migrated          bool              `json:"migrated" yaml:"migrated"` // Only used for checking if config has migrate in meta
	ApplicationConfig ApplicationConfig `json:"application" yaml:"application"`
}

type ConfigProperties struct {
	Version string `json:"version" yaml:"version"`
	EnvFile string `json:"envFile" yaml:"envFile"`
}

type ApplicationConfig struct {
	Name         string  `json:"name" yaml:"name,omitempty"`
	Type         string  `json:"manifestType" yaml:"manifestType,omitempty"`
	ResourcePath RelPath `json:"resourcePath" yaml:"resourcePath"`
	IgnoredPath  RelPath `json:"ignoredPath" yaml:"ignoredPath"`

	PreInstall  SortedRelPath `json:"onPreInstall" yaml:"onPreInstall"`
	PostInstall SortedRelPath `json:"onPostInstall" yaml:"onPostInstall"`
	PreUpgrade  SortedRelPath `json:"onPreUpgrade" yaml:"onPreUpgrade"`
	PostUpgrade SortedRelPath `json:"onPostUpgrade" yaml:"onPostUpgrade"`
	PreDelete   SortedRelPath `json:"onPreDelete" yaml:"onPreDelete"`
	PostDelete  SortedRelPath `json:"onPostDelete" yaml:"onPostDelete"`

	HelmValues     []*HelmValue       `json:"helmValues" yaml:"helmValues"`
	HelmVals       interface{}        `json:"helmVals" yaml:"helmVals"`
	HelmVersion    string             `json:"helmVersion" yaml:"helmVersion"`
	Env            []*Env             `json:"env" yaml:"env"`
	EnvFrom        EnvFrom            `json:"envFrom,omitempty" yaml:"envFrom,omitempty"`
	ServiceConfigs []*ServiceConfigV2 `json:"services" yaml:"services,omitempty"`
}

type HubConfig struct {
	Image string `json:"image" yaml:"image"`
}

type ContainerInstallConfig struct {
	Env         []*Env   `json:"env" yaml:"env"`
	EnvFrom     EnvFrom  `json:"envFrom" yaml:"envFrom"`
	PortForward []string `json:"portForward" yaml:"portForward"`
}

type ContainerDevConfig struct {
	GitUrl                string                 `json:"gitUrl" yaml:"gitUrl"`
	Image                 string                 `json:"image" yaml:"image"`
	Shell                 string                 `json:"shell" yaml:"shell"`
	WorkDir               string                 `json:"workDir" yaml:"workDir"`
	StorageClass          string                 `json:"storageClass" yaml:"storageClass"`
	DevContainerResources *ResourceQuota         `json:"resources" yaml:"resources"`
	PersistentVolumeDirs  []*PersistentVolumeDir `json:"persistentVolumeDirs" yaml:"persistentVolumeDirs"`
	Command               *DevCommands           `json:"command" yaml:"command"`
	DebugConfig           *DebugConfig           `json:"debug" yaml:"debug"`
	HotReload             bool                   `json:"hotReload" yaml:"hotReload"`
	UseDevContainer       bool                   `json:"useDevContainer,omitempty" yaml:"useDevContainer,omitempty"`
	Sync                  *SyncConfig            `json:"sync" yaml:"sync"`
	Env                   []*Env                 `json:"env" yaml:"env"`
	EnvFrom               *EnvFrom               `json:"envFrom,omitempty" yaml:"envFrom,omitempty"`
	PortForward           []string               `json:"portForward" yaml:"portForward"`
	SidecarImage          string                 `json:"sidecarImage,omitempty" yaml:"sidecarImage,omitempty"`
}

type DevCommands struct {
	Build          []string `json:"build,omitempty" yaml:"build,omitempty"`
	Run            []string `json:"run" yaml:"run"`
	Debug          []string `json:"debug" yaml:"debug"`
	HotReloadRun   []string `json:"hotReloadRun,omitempty" yaml:"hotReloadRun,omitempty"`
	HotReloadDebug []string `json:"hotReloadDebug,omitempty" yaml:"hotReloadDebug,omitempty"`
}

type SyncConfig struct {
	Type              string   `json:"type" yaml:"type"`
	Mode              string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	FilePattern       []string `json:"filePattern" yaml:"filePattern"`
	IgnoreFilePattern []string `json:"ignoreFilePattern" yaml:"ignoreFilePattern"`
}

type DebugConfig struct {
	RemoteDebugPort int `json:"remoteDebugPort" yaml:"remoteDebugPort"`
}

type DependLabelSelector struct {
	Pods []string `json:"pods" yaml:"pods"`
	Jobs []string `json:"jobs" yaml:"jobs"`
}

type HelmValue struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
}

type Env struct {
	Name  string `json:"name" yaml:"name"`
	Value string `json:"value" yaml:"value"`
}

type EnvFrom struct {
	EnvFile []*EnvFile `json:"envFile" yaml:"envFile"`
}

type EnvFile struct {
	Path string `json:"path" yaml:"path"`
}

func (n *NocalHostAppConfigV2) GetSvcConfigV2(svcName string, svcType base.SvcType) *ServiceConfigV2 {
	if n != nil {
		for _, config := range n.ApplicationConfig.ServiceConfigs {
			if config.Name == svcName && base.SvcTypeOf(config.Type) == svcType {
				return config
			}
		}
	}
	return nil
}

// GetSvcConfigS If ServiceConfig not found, return a default one
func (n *NocalHostAppConfigV2) GetSvcConfigS(svcName string, svcType base.SvcType) ServiceConfigV2 {
	for _, config := range n.ApplicationConfig.ServiceConfigs {
		if config.Name == svcName && base.SvcTypeOf(config.Type) == svcType {
			return *config
		}
	}
	return ServiceConfigV2{Name: svcName, Type: string(svcType)}
}

func (n *NocalHostAppConfigV2) SetSvcConfigV2(svcConfig ServiceConfigV2) {
	if svcConfig.Name == "" || svcConfig.Type == "" {
		return
	}
	foundIndex := -1
	for index, config := range n.ApplicationConfig.ServiceConfigs {
		if config.Name == svcConfig.Name && config.Type == svcConfig.Type {
			foundIndex = index
			break
		}
	}
	if foundIndex >= 0 {
		n.ApplicationConfig.ServiceConfigs[foundIndex] = &svcConfig
		return
	}
	n.ApplicationConfig.ServiceConfigs = append(n.ApplicationConfig.ServiceConfigs, &svcConfig)
}

func (n *NocalHostAppConfigV2) FindSvcConfigInHub(svcName string, svcType base.SvcType, container, image string) *ServiceConfigV2 {
	svcConfig := n.GetSvcConfigS(svcName, svcType)
	if isSvcConfigInHubMatch(&svcConfig, container, image) {
		return &svcConfig
	}
	return nil
}

func isSvcConfigInHubMatch(svcConfig *ServiceConfigV2, container, image string) bool {
	if svcConfig == nil {
		return false
	}

	for _, c := range svcConfig.ContainerConfigs {
		if c.Name == container {
			if c.Hub == nil {
				return false
			}
			if !strings.Contains(image, c.Hub.Image) {
				return false
			}
			return true
		}
	}
	return false
}

func (c *ApplicationConfig) LoadManifests(tmpDir *fp.FilePathEnhance) []string {
	if c == nil {
		return []string{}
	}

	return clientgoutils.LoadValidManifest(
		c.ResourcePath.Load(tmpDir),
		c.PreInstall.Load(tmpDir),
		c.PostInstall.Load(tmpDir),
		c.PreUpgrade.Load(tmpDir),
		c.PostUpgrade.Load(tmpDir),
		c.IgnoredPath.Load(tmpDir),
		c.PreDelete.Load(tmpDir),
		c.PostDelete.Load(tmpDir),
	)
}
