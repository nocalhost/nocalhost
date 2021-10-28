/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package profile

import (
	"nocalhost/internal/nhctl/fp"
	"sort"
	"strconv"
)

// Deprecated
type NocalHostAppConfig struct {
	PreInstall   SortedRelPath        `json:"onPreInstall" yaml:"onPreInstall"`
	ResourcePath RelPath              `json:"resourcePath" yaml:"resourcePath"`
	SvcConfigs   []*ServiceDevOptions `json:"services" yaml:"services"`
	Name         string               `json:"name" yaml:"name"`
	Type         string               `json:"manifestType" yaml:"manifestType"`
	IgnoredPath  RelPath              `json:"ignoredPath" yaml:"ignoredPath"`
}

type PersistentVolumeDir struct {
	Path                 string `json:"path" yaml:"path"`
	Capacity             string `validate:"Quantity" json:"capacity,omitempty" yaml:"capacity,omitempty"`
	WaitForFirstConsumer bool   `json:"waitForFirstConsumer,omitempty" yaml:"waitForFirstConsumer,omitempty"`
}

type ResourceQuota struct {
	Limits   *QuotaList `json:"limits" yaml:"limits"`
	Requests *QuotaList `json:"requests" yaml:"requests"`
}

type QuotaList struct {
	Memory string `validate:"Quantity" json:"memory" yaml:"memory"`
	Cpu    string `validate:"Quantity" json:"cpu" yaml:"cpu"`
}

type ServiceDevOptions struct {
	Name                  string                 `json:"name" yaml:"name"`
	Type                  string                 `json:"serviceType" yaml:"serviceType"`
	GitUrl                string                 `json:"gitUrl" yaml:"gitUrl"`
	DevImage              string                 `json:"devContainerImage" yaml:"devContainerImage"`
	WorkDir               string                 `json:"workDir" yaml:"workDir"`
	Sync                  []string               `json:"syncDirs" yaml:"syncDirs,omitempty"` // dev start -s
	PriorityClass         string                 `json:"priorityClass,omitempty" yaml:"priorityClass,omitempty"`
	PersistentVolumeDirs  []*PersistentVolumeDir `json:"persistentVolumeDirs" yaml:"persistentVolumeDirs"`
	BuildCommand          []string               `json:"buildCommand,omitempty" yaml:"buildCommand,omitempty"`
	RunCommand            []string               `json:"runCommand,omitempty" yaml:"runCommand,omitempty"`
	DebugCommand          []string               `json:"debugCommand,omitempty" yaml:"debugCommand,omitempty"`
	HotReloadRunCommand   []string               `json:"hotReloadRunCommand,omitempty" yaml:"hotReloadRunCommand,omitempty"`
	HotReloadDebugCommand []string               `json:"hotReloadDebugCommand,omitempty" yaml:"hotReloadDebugCommand,omitempty"`
	DevContainerShell     string                 `json:"devContainerShell" yaml:"devContainerShell"`
	DevContainerResources *ResourceQuota         `json:"devContainerResources" yaml:"devContainerResources"`
	DevPort               []string               `json:"devPorts" yaml:"devPorts"`
	Jobs                  []string               `json:"dependJobsLabelSelector" yaml:"dependJobsLabelSelector,omitempty"`
	Pods                  []string               `json:"dependPodsLabelSelector" yaml:"dependPodsLabelSelector,omitempty"`
	SyncedPattern         []string               `json:"syncFilePattern" yaml:"syncFilePattern"`
	IgnoredPattern        []string               `json:"ignoreFilePattern" yaml:"ignoreFilePattern"`
}

type WeightablePath struct {
	Path   string `json:"path" yaml:"path"`
	Weight string `json:"weight" yaml:"weight"`
}

type NocalhostResource interface {
	Load(resourceDir string) []string
}

type SortedRelPath []*WeightablePath

func (c *SortedRelPath) Load(fp *fp.FilePathEnhance) []string {
	result := make([]string, 0)
	if c != nil {
		sort.Sort(c)
		for _, item := range *c {
			file := fp.RelOrAbs(item.Path)
			if err := file.CheckExist(); err != nil {
				continue
			}
			result = append(result, file.Abs())
		}
	}
	return result
}

type RelPath []string

func (c *RelPath) Load(fp *fp.FilePathEnhance) []string {
	result := make([]string, 0)
	if c != nil {
		for _, item := range *c {
			file := fp.RelOrAbs(item)
			if err := file.CheckExist(); err != nil {
				continue
			}
			result = append(result, file.Abs())
		}
	}
	return result
}

func (a SortedRelPath) Len() int      { return len(a) }
func (a SortedRelPath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortedRelPath) Less(i, j int) bool {
	iW, err := strconv.Atoi(a[i].Weight)
	if err != nil {
		iW = 0
	}

	jW, err := strconv.Atoi(a[j].Weight)
	if err != nil {
		jW = 0
	}
	return iW < jW
}
