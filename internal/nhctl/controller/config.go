/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"nocalhost/internal/nhctl/profile"
)

// GetConfig The result will not be nil
func (c *Controller) GetConfig() (*profile.ServiceConfigV2, error) {
	svcConfig := c.GetAppConfig().GetSvcConfigS(c.Name, c.Type)
	return &svcConfig, nil
}

func (c *Controller) GetAppConfig() *profile.NocalHostAppConfigV2 {
	return c.AppMeta.Config
}

func (c *Controller) UpdateConfig(config profile.ServiceConfigV2) error {
	c.AppMeta.Config.SetSvcConfigV2(config)
	return c.AppMeta.Update()
}

func (c *Controller) GetWorkDir(container string) string {
	svcConfig, _ := c.GetConfig()
	devConfig := svcConfig.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.WorkDir != "" {
		return devConfig.WorkDir
	}
	return profile.DefaultWorkDir
}

func (c *Controller) GetStorageClass(container string) string {
	svcProfile, _ := c.GetConfig()
	devConfig := svcProfile.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.StorageClass != "" {
		return devConfig.StorageClass
	}
	return ""
}

func (c *Controller) GetDevImage(container string) string {
	svcProfile, _ := c.GetConfig()
	devConfig := svcProfile.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.Image != "" {
		return devConfig.Image
	}
	return profile.DefaultDevImage
}

func (c *Controller) GetPersistentVolumeDirs(container string) []*profile.PersistentVolumeDir {
	svcProfile, _ := c.GetConfig()
	devConfig := svcProfile.GetContainerDevConfigOrDefault(container)
	if devConfig != nil {
		return devConfig.PersistentVolumeDirs
	}
	return nil
}
