/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"nocalhost/internal/nhctl/profile"

	corev1 "k8s.io/api/core/v1"
)

// GetConfig The result will not be nil
//func (c *Controller) GetConfig() (*profile.ServiceConfigV2, error) {
//	svcConfig := c.GetAppConfig().GetSvcConfigS(c.Name, c.Type)
//	return &svcConfig, nil
//}

func (c *Controller) Config() *profile.ServiceConfigV2 {
	return c.config
}

func (c *Controller) ReloadConfig() profile.ServiceConfigV2 {
	config := c.AppMeta.Config.GetSvcConfigS(c.Name, c.Type)
	c.config = &config
	return config
}

func (c *Controller) GetAppConfig() *profile.NocalHostAppConfigV2 {
	return c.AppMeta.Config
}

func (c *Controller) UpdateConfig(config profile.ServiceConfigV2) error {
	config.Name = c.Name
	config.Type = string(c.Type)
	c.AppMeta.Config.SetSvcConfigV2(config)
	if err := c.AppMeta.Update(); err != nil {
		return err
	}
	c.config = &config
	return nil
}

func (c *Controller) GetWorkDir(container string) string {
	devConfig := c.config.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.WorkDir != "" {
		return devConfig.WorkDir
	}
	return profile.DefaultWorkDir
}

func (c *Controller) GetStorageClass(container string) string {
	devConfig := c.config.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.StorageClass != "" {
		return devConfig.StorageClass
	}
	return ""
}

func (c *Controller) GetDevImage(container string) string {
	devConfig := c.config.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.Image != "" {
		return devConfig.Image
	}
	return profile.DefaultDevImage
}

func (c *Controller) GetPersistentVolumeDirs(container string) []*profile.PersistentVolumeDir {
	devConfig := c.config.GetContainerDevConfigOrDefault(container)
	if devConfig != nil {
		return devConfig.PersistentVolumeDirs
	}
	return nil
}

func (c *Controller) GetImagePullPolicy(container string) corev1.PullPolicy {
	devConfig := c.config.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.Image != "" {
		return devConfig.ImagePullPolicy
	}
	return corev1.PullIfNotPresent
}
