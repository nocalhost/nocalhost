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

func (a *Application) CheckIfSvcIsDeveloping(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.Developing
}

func (a *Application) CheckIfSvcIsSyncthing(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.Syncing
}

func (a *Application) CheckIfSvcIsPortForwarded(svcName string) bool {
	profile := a.GetSvcProfile(svcName)
	if profile == nil {
		return false
	}
	return profile.PortForwarded
}

func (a *Application) GetSvcProfile(svcName string) *SvcProfile {
	//if a.AppProfile == nil {
	//	return nil
	//}
	if a.AppProfile.SvcProfile == nil {
		return nil
	}
	for _, svcProfile := range a.AppProfile.SvcProfile {
		if svcProfile.ActualName == svcName {
			return svcProfile
		}
	}
	return nil
}

//func (a *Application) UpdateSvcProfile(svcName string) {
//
//}
