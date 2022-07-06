/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package applications

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// Create Delete Application
// @Summary Delete Application
// @Description The user deletes the application, and also deletes the configured development space in the application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id} [delete]
func Delete(c *gin.Context) {
	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))

	get, err := service.Svc.ApplicationSvc.Get(c, applicationId)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationDelete, nil)
		return
	}
	if !ginbase.IsAdmin(c) && !ginbase.IsCurrentUser(c, get.UserId) {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	// delete application database record
	err = service.Svc.ApplicationSvc.Delete(c, applicationId)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationDelete, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
