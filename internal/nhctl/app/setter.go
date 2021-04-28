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

import (
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/profile"
)

func (a *Application) SetPortForwardedStatus(svcName string, is bool) error {
	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.SvcProfileV2(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.PortForwarded = is
	return profileV2.Save()
}

func (a *Application) SetSyncingStatus(svcName string, is bool) error {
	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.SvcProfileV2(svcName)
	if svcProfile == nil {
		return errors.New("Failed to get svc profile")
	}
	svcProfile.Syncing = is
	return profileV2.Save()
}

func (a *Application) SetSyncthingPort(svcName string, remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.SvcProfileV2(svcName)
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
	profileV2, err := profile.NewAppProfileV2ForUpdate(a.NameSpace, a.Name)
	if err != nil {
		return err
	}
	defer profileV2.CloseDb()

	svcProfile := profileV2.SvcProfileV2(svcName)
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
