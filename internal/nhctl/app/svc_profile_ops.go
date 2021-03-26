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

import (
	"nocalhost/internal/nhctl/profile"
)

func (a *Application) CheckIfSvcIsDeveloping(svcName string) bool {
	profileV2, _ := profile.NewAppProfileV2(a.NameSpace, a.Name, true)
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return false
	}

	return svcProfile.Developing
}

func (a *Application) CheckIfSvcIsSyncthing(svcName string) bool {
	profileV2, _ := profile.NewAppProfileV2(a.NameSpace, a.Name, true)
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return false
	}
	return svcProfile.Syncing
}

func (a *Application) CheckIfSvcIsPortForwarded(svcName string) bool {
	profileV2, _ := profile.NewAppProfileV2(a.NameSpace, a.Name, true)
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return false
	}
	return svcProfile.PortForwarded
}

//func (a *Application) GetSvcProfileV2(svcName string) *profile.SvcProfileV2 {
//
//	for _, svcProfile := range a.AppProfileV2.SvcProfile {
//		if svcProfile.ActualName == svcName {
//			return svcProfile
//		}
//	}
//
//	// If not profile found, init one
//	if a.AppProfileV2.SvcProfile == nil {
//		a.AppProfileV2.SvcProfile = make([]*profile.SvcProfileV2, 0)
//	}
//	svcProfile := &profile.SvcProfileV2{
//		ServiceConfigV2: &profile.ServiceConfigV2{
//			Name: svcName,
//			Type: string(Deployment),
//			ContainerConfigs: []*profile.ContainerConfig{
//				{
//					Dev: &profile.ContainerDevConfig{
//						Image:   profile.DefaultDevImage,
//						WorkDir: profile.DefaultWorkDir,
//					},
//				},
//			},
//		},
//		ActualName: svcName,
//	}
//	a.AppProfileV2.SvcProfile = append(a.AppProfileV2.SvcProfile, svcProfile)
//
//	err := a.SaveProfile()
//	if err != nil {
//		log.WarnE(err, "")
//	}
//
//	return svcProfile
//}

//func (a *Application) GetSvcProfile(svcName string) *SvcProfile {
//	//if a.AppProfile == nil {
//	//	return nil
//	//}
//	if a.AppProfile.SvcProfile == nil {
//		return nil
//	}
//	for _, svcProfile := range a.AppProfile.SvcProfile {
//		if svcProfile.ActualName == svcName {
//			return svcProfile
//		}
//	}
//	// If not profile found, init one
//	svcProfile := &SvcProfile{
//		ServiceDevOptions: &ServiceDevOptions{
//			Name:     svcName,
//			Type:     Deployment,
//			DevImage: DefaultDevImage,
//			WorkDir:  DefaultWorkDir,
//		},
//		ActualName:                             svcName,
//		Developing:                             false,
//		PortForwarded:                          false,
//		Syncing:                                false,
//		RemoteSyncthingPort:                    0,
//		RemoteSyncthingGUIPort:                 0,
//		SyncthingSecret:                        "",
//		LocalSyncthingPort:                     0,
//		LocalSyncthingGUIPort:                  0,
//		LocalAbsoluteSyncDirFromDevStartPlugin: nil,
//		DevPortList:                            nil,
//		PortForwardStatusList:                  nil,
//		PortForwardPidList:                     nil,
//		SyncedPatterns:                         nil,
//		IgnoredPatterns:                        nil,
//	}
//	a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcProfile)
//	a.SaveProfile()
//	return svcProfile
//}

//func (a *Application) UpdateSvcProfile(svcName string) {
//
//}
