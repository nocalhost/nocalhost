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
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/profile"
)

//func (a *Application) SetRemoteSyncthingGUIPort(svcName string, port int) error {
//	a.GetSvcProfileV2(svcName).RemoteSyncthingGUIPort = port
//	return a.SaveProfile()
//}

//func (a *Application) SetLocalSyncthingPort(svcName string, port int) error {
//	a.GetSvcProfileV2(svcName).LocalSyncthingPort = port
//	return a.SaveProfile()
//}

//func (a *Application) SetLocalSyncthingGUIPort(svcName string, port int) error {
//	a.GetSvcProfileV2(svcName).LocalSyncthingGUIPort = port
//	return a.SaveProfile()
//}

func (a *Application) SetDevelopingStatus(svcName string, is bool) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.Developing = is

	//a.GetSvcProfileV2(svcName).Developing = is
	return profileV2.Save()
}

func (a *Application) SetAppType(t AppType) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	profileV2.AppType = string(t)
	//a.AppProfileV2.AppType = string(t)
	return profileV2.Save()
}

func (a *Application) SetPortForwardedStatus(svcName string, is bool) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.PortForwarded = is
	return profileV2.Save()
}

//func (a *Application) SetRemoteSyncthingPort(svcName string, port int) error {
//	a.GetSvcProfileV2(svcName).RemoteSyncthingPort = port
//	return a.SaveProfile()
//}

func (a *Application) SetSyncingStatus(svcName string, is bool) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.Syncing = is
	//a.GetSvcProfileV2(svcName).Syncing = is
	return profileV2.Save()
}

func (a *Application) SetDevEndProfileStatus(svcName string) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.Developing = false
	//a.GetSvcProfileV2(svcName).Developing = false
	return profileV2.Save()
}

func (a *Application) SetSyncthingPort(svcName string, remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.RemoteSyncthingPort = remotePort
	svcProfile.RemoteSyncthingGUIPort = remoteGUIPort
	svcProfile.LocalSyncthingPort = localPort
	svcProfile.LocalSyncthingGUIPort = localGUIPort
	return profileV2.Save()
}

func (a *Application) SetSyncthingProfileEndStatus(svcName string) error {
	profileV2, err := profile.NewAppProfileV2(a.NameSpace, a.Name, false)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.FetchSvcProfileV2FromProfile(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.RemoteSyncthingPort = 0
	svcProfile.RemoteSyncthingGUIPort = 0
	svcProfile.LocalSyncthingPort = 0
	svcProfile.LocalSyncthingGUIPort = 0
	svcProfile.PortForwarded = false
	svcProfile.Syncing = false
	svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = []string{}
	return profileV2.Save()
}
