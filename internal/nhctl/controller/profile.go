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

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

func (c *Controller) SaveConfigToProfile(config *profile.ServiceConfigV2) error {
	return c.UpdateProfile(
		func(profileV2 *profile.AppProfileV2, svcPro *profile.SvcProfileV2) error {
			if svcPro != nil {
				config.Name = c.Name
				svcPro.ServiceConfigV2 = config
			} else {
				config.Name = c.Name
				svcPro = &profile.SvcProfileV2{
					ServiceConfigV2: config,
					ActualName:      c.Name,
				}
				profileV2.SvcProfile = append(profileV2.SvcProfile, svcPro)
			}
			return nil
		},
	)
}

func (c *Controller) GetAppProfile() (*profile.AppProfileV2, error) {
	return nocalhost.GetProfileV2(c.NameSpace, c.AppName)
}

func (c *Controller) GetProfile() (*profile.SvcProfileV2, error) {
	p, err := c.GetAppProfile()
	if err != nil {
		return nil, err
	}
	return p.SvcProfileV2(c.Name, string(c.Type)), nil
}

func (c *Controller) GetWorkDir(container string) string {
	svcProfile, _ := c.GetProfile()
	if svcProfile != nil && svcProfile.GetContainerDevConfigOrDefault(container).WorkDir != "" {
		return svcProfile.GetContainerDevConfigOrDefault(container).WorkDir
	}
	return profile.DefaultWorkDir
}

func (c *Controller) GetStorageClass(container string) string {
	svcProfile, _ := c.GetProfile()
	if svcProfile != nil && svcProfile.GetContainerDevConfigOrDefault(container).StorageClass != "" {
		return svcProfile.GetContainerDevConfigOrDefault(container).StorageClass
	}
	return ""
}

func (c *Controller) GetDevImage(container string) string {
	svcProfile, _ := c.GetProfile()
	if svcProfile != nil && svcProfile.GetContainerDevConfigOrDefault(container).Image != "" {
		return svcProfile.GetContainerDevConfigOrDefault(container).Image
	}
	return profile.DefaultDevImage
}

func (c *Controller) GetPortForwardForSync() (*profile.DevPortForward, error) {
	var err error
	svcProfile, err := c.GetProfile()
	if err != nil {
		return nil, err
	}
	for _, pf := range svcProfile.DevPortForwardList {
		if pf.Role == "SYNC" {
			return pf, nil
		}
	}
	return nil, nil
}

func (c *Controller) SetPortForwardedStatus(is bool) error {
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to get controller profile")
			}
			svcProfile.PortForwarded = is
			return nil
		},
	)
}

func (c *Controller) setSyncthingProfileEndStatus() error {
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to get controller profile")
			}
			svcProfile.RemoteSyncthingPort = 0
			svcProfile.RemoteSyncthingGUIPort = 0
			svcProfile.LocalSyncthingPort = 0
			svcProfile.LocalSyncthingGUIPort = 0
			svcProfile.PortForwarded = false
			svcProfile.Syncing = false
			svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin = []string{}
			return nil
		},
	)
}

// You should `CheckIfPortForwardExists` before adding a port-forward to db
func (c *Controller) AddPortForwardToDB(port *profile.DevPortForward) error {
	return c.UpdateProfile(
		func(profileV2 *profile.AppProfileV2, svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to get controller profile")
			}

			svcProfile.DevPortForwardList = append(svcProfile.DevPortForwardList, port)
			return nil
		},
	)
}

func (c *Controller) DeletePortForwardFromDB(localPort, remotePort int) error {

	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to add a port-forward to db")
			}

			indexToDelete := -1
			for index, portForward := range svcProfile.DevPortForwardList {
				if portForward.LocalPort == localPort && portForward.RemotePort == remotePort {
					indexToDelete = index
					break
				}
			}

			// remove portForward from DevPortForwardList
			if indexToDelete > -1 {
				originList := svcProfile.DevPortForwardList
				newList := make([]*profile.DevPortForward, 0)
				for index, p := range originList {
					if index != indexToDelete {
						newList = append(newList, p)
					}
				}
				svcProfile.DevPortForwardList = newList
				return nil
			}

			errMsg := fmt.Sprintf("Port forward %d-%d not found", localPort, remotePort)
			log.Log(errMsg)
			return errors.New(errMsg)
		},
	)
}

func (c *Controller) SetSyncthingPort(remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			if svcProfile == nil {
				return errors.New("Failed to get controller profile")
			}
			svcProfile.RemoteSyncthingPort = remotePort
			svcProfile.RemoteSyncthingGUIPort = remoteGUIPort
			svcProfile.LocalSyncthingPort = localPort
			svcProfile.LocalSyncthingGUIPort = localGUIPort
			return nil
		},
	)
}

func (c *Controller) associateLocalDir(associate string) error {
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			svcProfile.Associate = associate
			return nil
		},
	)
}

// You need to closeDB for profile explicitly
func (c *Controller) GetProfileForUpdate() (*profile.AppProfileV2, error) {
	return profile.NewAppProfileV2ForUpdate(c.NameSpace, c.AppName)
}
