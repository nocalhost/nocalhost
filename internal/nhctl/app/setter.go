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

func (a *Application) SetRemoteSyncthingGUIPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalSyncthingPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).LocalSyncthingPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalSyncthingGUIPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetDevelopingStatus(svcName string, is bool) error {
	a.GetSvcProfile(svcName).Developing = is
	return a.AppProfile.Save()
}

func (a *Application) SetAppType(t AppType) error {
	a.AppProfile.AppType = t
	return a.AppProfile.Save()
}

func (a *Application) SetPortForwardedStatus(svcName string, is bool) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	a.GetSvcProfile(svcName).PortForwarded = is
	return a.AppProfile.Save()
}

func (a *Application) SetRemoteSyncthingPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetSyncingStatus(svcName string, is bool) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		return err
	}
	a.GetSvcProfile(svcName).Syncing = is
	return a.AppProfile.Save()
}

func (a *Application) SetDevEndProfileStatus(svcName string) error {
	a.GetSvcProfile(svcName).Developing = false
	return a.AppProfile.Save()
}

func (a *Application) SetSyncthingPort(svcName string, remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = remotePort
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = remoteGUIPort
	a.GetSvcProfile(svcName).LocalSyncthingPort = localPort
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = localGUIPort
	return a.AppProfile.Save()
}

func (a *Application) SetSyncthingProfileEndStatus(svcName string) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = 0
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = 0
	a.GetSvcProfile(svcName).LocalSyncthingPort = 0
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = 0
	a.GetSvcProfile(svcName).PortForwarded = false
	a.GetSvcProfile(svcName).Syncing = false
	a.GetSvcProfile(svcName).DevPortList = []string{}
	a.GetSvcProfile(svcName).LocalAbsoluteSyncDirFromDevStartPlugin = []string{}
	return a.AppProfile.Save()
}
