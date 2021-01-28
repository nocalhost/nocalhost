/*
Copyright 2020 The Nocalhost Authors.
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

import "path/filepath"

func (a *Application) GetDependencies() []*SvcDependency {
	result := make([]*SvcDependency, 0)

	if a.config == nil {
		return nil
	}

	svcConfigs := a.config.SvcConfigs
	if svcConfigs == nil || len(svcConfigs) == 0 {
		return nil
	}

	for _, svcConfig := range svcConfigs {
		if svcConfig.Pods == nil && svcConfig.Jobs == nil {
			continue
		}
		svcDep := &SvcDependency{
			Name: svcConfig.Name,
			Type: string(svcConfig.Type),
			Jobs: svcConfig.Jobs,
			Pods: svcConfig.Pods,
		}
		result = append(result, svcDep)
	}
	return result
}

// Get local path of resource dirs
// If resource path undefined, use git url
func (a *Application) GetResourceDir() []string {
	var resourcePath []string
	if len(a.AppProfile.ResourcePath) != 0 {
		for _, path := range a.AppProfile.ResourcePath {
			fullPath := filepath.Join(a.getGitDir(), path)
			resourcePath = append(resourcePath, fullPath)
		}
		return resourcePath
	}
	return []string{a.getGitDir()}
}

func (a *Application) getIgnoredPath() []string {
	results := make([]string, 0)
	for _, path := range a.AppProfile.IgnoredPath {
		results = append(results, filepath.Join(a.getGitDir(), path))
	}
	return results
}

func (a *Application) GetDefaultWorkDir(svcName string) string {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.WorkDir != "" {
		return svcProfile.WorkDir
	}
	return DefaultWorkDir
}

func (a *Application) GetPersistentVolumeDirs(svcName string) []*PersistentVolumeDir {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil {
		return svcProfile.PersistentVolumeDirs
	}
	return nil
}

func (a *Application) GetDefaultSideCarImage(svcName string) string {
	return DefaultSideCarImage
}

func (a *Application) GetDefaultDevImage(svcName string) string {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.DevImage != "" {
		return svcProfile.DevImage
	}
	return DefaultDevImage
}

func (a *Application) GetDefaultDevPort(svcName string) []string {
	config := a.GetSvcProfile(svcName)
	if config != nil && len(config.DevPort) > 0 {
		return config.DevPort
	}
	return []string{}
}
