/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Update Update user information
// @Summary Update user information (including disabled users)
// @Description Update a user by ID，Only status is required
// @Tags Users
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "The user's database id index num"
// @Param user body user.UpdateUserRequest true "Update user info"
// @Success 200 {object} model.UserBaseModel
// @Router /v1/users/{id} [put]
func Update(c *gin.Context) {
	// Get the user id from the url parameter.
	userId := cast.ToUint64(c.Param("id"))

	// Binding the user data.
	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		log.Warnf("bind request param err: %+v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	userMap := model.UserBaseModel{}
	if len(req.Email) > 0 {
		userMap.Email = req.Email
	}
	if len(req.Name) > 0 {
		userMap.Name = req.Name
	}
	if len(req.Password) > 0 {
		pwd, err := auth.Encrypt(req.Password)
		if err != nil {
			api.SendResponse(c, errno.InternalServerError, nil)
			return
		}
		userMap.Password = pwd
	}

	// Only administrator can modify status and isAdmin fields
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) == 1 {
		if req.IsAdmin != nil {
			userMap.IsAdmin = req.IsAdmin
		}
		if req.Status != nil {
			userMap.Status = req.Status
		}
	} else {
		uid, _ := c.Get("userId")
		if cast.ToUint64(uid) != userId {
			api.SendResponse(c, errno.ErrPermissionDenied, nil)
			return
		}
	}

	result, err := service.Svc.UserSvc().UpdateUser(context.TODO(), userId, &userMap)
	if err != nil {
		log.Warnf("[user] update user err, %v", err)
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	api.SendResponse(c, nil, result)
}
