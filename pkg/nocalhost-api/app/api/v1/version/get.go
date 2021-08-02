/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package version

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
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
