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

package profile

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// Deprecated: this struct is deprecated
type AppProfile struct {
	path                    string
	Name                    string        `json:"name" yaml:"name"`
	ChartName               string        `json:"chart_name" yaml:"chartName,omitempty"` // This name may come from config.yaml or --helm-chart-name
	ReleaseName             string        `json:"release_name yaml:releaseName"`
	Namespace               string        `json:"namespace" yaml:"namespace"`
	Kubeconfig              string        `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string        `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 string        `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfile `json:"svc_profile" yaml:"svcProfile"` // This will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool          `json:"installed" yaml:"installed"`
	ResourcePath            []string      `json:"resource_path" yaml:"resourcePath"`
	IgnoredPath             []string      `json:"ignoredPath" yaml:"ignoredPath"`
}


type WeightablePath struct {
	Path   string `json:"path" yaml:"path"`
	Weight string `json:"weight" yaml:"weight"`
}

type NocalhostResource interface {
	Load(resourceDir string) []string
}

type SortedRelPath []*WeightablePath

func (s *SortedRelPath) Load(resourceDir string) []string {
	result := make([]string, 0)
	if s != nil {
		sort.Sort(s)
		for _, item := range *s {
			itemPath := filepath.Join(resourceDir, item.Path)
			if _, err2 := os.Stat(itemPath); err2 != nil {
				continue
			}
			result = append(result, itemPath)
		}
	}
	return result
}

type RelPath []string

func (c *RelPath) Load(resourceDir string) []string {
	result := make([]string, 0)
	if c != nil {
		for _, item := range *c {
			itemPath := filepath.Join(resourceDir, item)
			if _, err2 := os.Stat(itemPath); err2 != nil {
				continue
			}
			result = append(result, itemPath)
		}
	}
	return result
}

func (s SortedRelPath) Len() int      { return len(s) }
func (s SortedRelPath) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortedRelPath) Less(i, j int) bool {
	iW, err := strconv.Atoi(s[i].Weight)
	if err != nil {
		iW = 0
	}

	jW, err := strconv.Atoi(s[j].Weight)
	if err != nil {
		jW = 0
	}
	return iW < jW
}
