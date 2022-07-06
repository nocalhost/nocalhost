/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
)

// Register Add developer
// @Summary Add developer
// @Description Admin add developer
// @Tags Users
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param register body user.CreateUserRequest true "Reg user info"
// @Success 200 {object} model.UserInfo
// @Router /v1/users [post]
func Create(c *gin.Context) {
	// Binding the data with the u struct.
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("create user bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	if req.Password != req.ConfirmPassword {
		log.Warnf("twice password is not same")
		api.SendResponse(c, errno.ErrTwicePasswordNotMatch, nil)
		return
	}

	u, err := service.Svc.UserSvc.Create(c, req.Email, req.Password, req.Name, "", 0, req.Status, req.IsAdmin)
	if err != nil {
		log.Warnf("register err: %v", err)
		api.SendResponse(c, errno.ErrRegisterFailed, nil)
		return
	}

	api.SendResponse(c, nil, u)
}
