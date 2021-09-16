/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package app

import (
	"path/filepath"
)

func (a *Application) GetDependencies() []*SvcDependency {
	result := make([]*SvcDependency, 0)

	svcConfigs := a.appMeta.Config.ApplicationConfig.ServiceConfigs
	if len(svcConfigs) == 0 {
		return nil
	}

	for _, svcConfig := range svcConfigs {
		if svcConfig.DependLabelSelector == nil {
			continue
		}
		if svcConfig.DependLabelSelector.Pods == nil && svcConfig.DependLabelSelector.Jobs == nil {
			continue
		}
		svcDep := &SvcDependency{
			Name: svcConfig.Name,
			Type: svcConfig.Type,
			Jobs: svcConfig.DependLabelSelector.Jobs,
			Pods: svcConfig.DependLabelSelector.Pods,
		}
		result = append(result, svcDep)
	}
	return result
}

// Get local path of resource dirs
// If resource path undefined, use git url
func (a *Application) GetResourceDir(tmpDir string) []string {
	//appProfile, _ := a.GetProfile()
	appProfile := a.GetApplicationConfigV2()
	var resourcePath []string
	if len(appProfile.ResourcePath) != 0 {
		for _, path := range appProfile.ResourcePath {
			fullPath := filepath.Join(tmpDir, path)
			resourcePath = append(resourcePath, fullPath)
		}
		return resourcePath
	}
	return []string{tmpDir}
}

func (a *Application) getIgnoredPath() []string {
	//appProfile, _ := a.GetProfile()
	appProfile := a.GetApplicationConfigV2()
	results := make([]string, 0)
	for _, path := range appProfile.IgnoredPath {
		results = append(results, filepath.Join(a.ResourceTmpDir, path))
	}
	return results
}
