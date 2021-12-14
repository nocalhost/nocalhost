/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package applications

import (
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func PublicSwitch(c *gin.Context) {
	var req AppPublicSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("update application bind err: %s", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	//userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	app, err2 := service.Svc.ApplicationSvc().Get(c, applicationId)
	if err2 != nil {
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}
	if !ginbase.IsAdmin(c) && !ginbase.IsCurrentUser(c, app.UserId) {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	err := service.Svc.ApplicationSvc().PublicSwitch(c, applicationId, *req.Public)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}

// Create Edit application
// @Summary Edit application
// @Description Edit application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "Application ID"
// @Param CreateAppRequest body applications.CreateAppRequest true "The application info"
// @Success 200 {object} model.ApplicationModel
// @Router /v1/application/{id} [put]
func Update(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("update application bind err: %s", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	// adapt earlier version
	if req.Public == nil {
		u := uint8(1)
		req.Public = &u
	}

	// normal user can't not create public applications
	if !ginbase.IsAdmin(c) {
		deny := uint8(0)
		req.Public = &deny
	}

	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	model := model.ApplicationModel{
		ID: applicationId,
		// UserId:  userId.(uint64),
		Context: req.Context,
		Status:  *req.Status,
		Public:  *req.Public,
	}
	result, err := service.Svc.ApplicationSvc().Update(c, &model)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, result)
}
