/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package version

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"golang.org/x/mod/semver"

	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/registry"
)

// Get Get api server version
// @Summary Get api server version
// @Description Get api server version
// @Tags Version
// @Accept  json
// @Produce  json
// @Success 200 {string} json "{"code":0,"message":"OK","data":{"version":"","commit_id":"","branch":""}}"
// @Router /v1/version [get]
func Get(c *gin.Context) {
	version := map[string]string{
		"version":   global.Version,
		"commit_id": global.CommitId,
		"branch":    global.Branch,
	}
	api.SendResponse(c, errno.OK, version)
}

// UpgradeInfo api server version update info
// @Summary UpgradeInfo api server version update info
// @Description UpgradeInfo api server version update info
// @Tags Version
// @Accept  json
// @Produce  json
// @Success 200 {object} model.VersionUpgradeInfo "{"code":0,"message":"OK","data":model.VersionUpgradeInfo}"
// @Router /v1/nocalhost/version/upgrade_info [get]
func UpgradeInfo(c *gin.Context) {
	v, err := getVersionInfo()
	if err != nil {
		log.Error(err)
	}
	api.SendResponse(c, errno.OK, v)
}

func getVersionInfo() (model.VersionUpgradeInfo, error) {
	version := model.VersionUpgradeInfo{
		CurrentVersion: global.Version,
	}

	reg, err := registry.New(fmt.Sprintf("https://%s", global.NocalhostRegistry), "", "")
	if err != nil {
		return version, err
	}
	tags, err := reg.Tags(global.Nocalhostrepository)
	if err != nil {
		return version, err
	}

	v := global.Version
	for _, tag := range tags {
		if semver.IsValid(tag) && semver.IsValid(v) {
			fmt.Println(tag)
			if semver.Compare(v, tag) == -1 {
				v = tag
				version.UpgradeVersion = tag
				version.HasNewVersion = true
			}
		}
	}

	return version, nil
}
