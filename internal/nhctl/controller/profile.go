/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/hub"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

//func (c *Controller) SaveConfigToProfile(config *profile.ServiceConfigV2) error {
//	return c.UpdateProfile(
//		func(profileV2 *profile.AppProfileV2, svcPro *profile.SvcProfileV2) error {
//			if svcPro != nil {
//				config.Name = c.Name
//				svcPro.ServiceConfigV2 = config
//			} else {
//				config.Name = c.Name
//				svcPro = &profile.SvcProfileV2{
//					ServiceConfigV2: config,
//					ActualName:      c.Name,
//				}
//				profileV2.SvcProfile = append(profileV2.SvcProfile, svcPro)
//			}
//			return nil
//		},
//	)
//}

func (c *Controller) GetAppProfile() (*profile.AppProfileV2, error) {
	return nocalhost.GetProfileV2(c.NameSpace, c.AppName, c.AppMeta.NamespaceId)
}

func (c *Controller) IsPortForwarding() bool {
	v2, err := nocalhost.GetProfileV2(c.NameSpace, c.AppName, c.AppMeta.NamespaceId)
	if err != nil {
		return true
	}
	for _, profileV2 := range v2.SvcProfile {
		if len(profileV2.DevPortForwardList) != 0 {
			return true
		}
	}
	return false
}

func (c *Controller) GetProfile() (*profile.SvcProfileV2, error) {
	p, err := c.GetAppProfile()
	if err != nil {
		return nil, err
	}
	return p.SvcProfileV2(c.Name, string(c.Type)), nil
}
func (c *Controller) LoadConfigFromHub() error {
	containers, err := c.GetContainers()
	if err != nil {
		return err
	}
	for _, container := range containers {
		_ = c.LoadConfigFromHubC(container.Name)
	}
	return nil
}

func (c *Controller) LoadConfigFromHubC(container string) error {
	p := c.Config()

	for _, cc := range p.ContainerConfigs {
		if cc != nil && cc.Dev != nil && cc.Dev.Image != "" {
			return nil
		}
	}

	cc := p.GetContainerConfig(container)
	if cc == nil || cc.Dev == nil || cc.Dev.Image == "" {
		log.Logf("%s config not found, try to load it from hub", container)
		originImage, err := c.GetContainerImage(container)
		if err == nil {
			// load config from hub
			svcConfig, err := hub.FindNocalhostSvcConfig(c.AppName, c.Name, c.Type, container, originImage)
			if err != nil {
				log.LogE(err)
			}
			if svcConfig != nil {
				svcConfig.Name = c.Name
				svcConfig.Type = string(c.Type)
				if err := c.UpdateConfig(*svcConfig); err != nil {
					log.Logf("Load nocalhost svc config from hub fail, fail while updating svc profile, err: %s", err.Error())
				}
			}
		}
	}
	return nil
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
			svcProfile.PortForwarded = is
			return nil
		},
	)
}

func (c *Controller) setSyncthingProfileEndStatus() error {
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
			//if duplicateDevMode {
			//	svcProfile.DuplicateDevModeSyncthingSecretName = ""
			//} else {
			//	svcProfile.SyncthingSecret = ""
			//}
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
	return c.UpdateSvcProfile(
		func(svcProfile *profile.SvcProfileV2) error {
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

// GetProfileForUpdate You need to closeDB for profile explicitly
func (c *Controller) GetProfileForUpdate() (*profile.AppProfileV2, error) {
	return profile.NewAppProfileV2ForUpdate(c.NameSpace, c.AppName, c.AppMeta.NamespaceId)
}

func UpdateSvcConfig(ns, appName, kubeconfig string, config *profile.ServiceConfigV2) error {
	meta, err := nocalhost.GetApplicationMeta(appName, ns, kubeconfig)
	if err != nil {
		return err
	}
	if !meta.IsInstalled() {
		return errors.New(fmt.Sprintf("AppMeta %s-%s is not installed", appName, ns))
	}
	meta.Config.SetSvcConfigV2(*config)
	return meta.Update()
}

func GetSvcConfig(ns, appName, svcName, kubeconfig string, svcType base.SvcType) (*profile.ServiceConfigV2, error) {
	meta, err := nocalhost.GetApplicationMeta(appName, ns, kubeconfig)
	if err != nil {
		return nil, err
	}
	if !meta.IsInstalled() {
		return nil, errors.New(fmt.Sprintf("AppMeta %s-%s is not installed", appName, ns))
	}
	return meta.Config.GetSvcConfigV2(svcName, svcType), nil
}
