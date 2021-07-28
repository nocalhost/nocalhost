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

//// Get local path of resource dirs
//// If resource path undefined, use git url
//func (a *Application) GetResourceDir(tmpDir string) []string {
//	appProfile, _ := a.GetProfile()
//	var resourcePath []string
//	if len(appProfile.ResourcePath) != 0 {
//		for _, path := range appProfile.ResourcePath {
//			fullPath := filepath.Join(tmpDir, path)
//			resourcePath = append(resourcePath, fullPath)
//		}
//		return resourcePath
//	}
//	return []string{tmpDir}
//}

//func (a *Application) getIgnoredPath() []string {
//	appProfile, _ := a.GetProfile()
//	results := make([]string, 0)
//	for _, path := range appProfile.IgnoredPath {
//		results = append(results, filepath.Join(a.ResourceTmpDir, path))
//	}
//	return results
//}
