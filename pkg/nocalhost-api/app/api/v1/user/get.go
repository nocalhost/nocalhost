/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"context"
	"github.com/spf13/cast"
	"strconv"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
)

// Get Get user details
// @Summary Get user details
// @Description Get user details
// @Tags Users
// @Accept  json
// @Produce  json
// @Param id path string true "Users ID"
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.UserInfo "Userinfo"
// @Router /v1/users/{id} [get]
func Get(c *gin.Context) {
	userID := cast.ToUint64(c.Param("id"))
	if userID == 0 {
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}

	// Get the user by the `user_id` from the database.
	u, err := service.Svc.UserSvc.GetUserByID(context.TODO(), userID)
	if err != nil {
		log.Warnf("get user info err: %v", err)
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	api.SendResponse(c, nil, u)
}

// Get Get user personal information
// @Summary Get user personal information
// @Description Get user personal information
// @Tags Users
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.UserInfo "Userinfo"
// @Router /v1/me [get]
func GetMe(c *gin.Context) {
	userID, _ := c.Get("userId")
	if userID == 0 {
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}

	// Get the user by the `user_id` from the database.
	u, err := service.Svc.UserSvc.GetUserByID(context.TODO(), userID.(uint64))
	if err != nil {
		log.Warnf("get user info err: %v", err)
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	api.SendResponse(c, nil, u)
}

// Get Get user list
// @Summary Get user list
// @Description Get userlist
// @Tags Users
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.UserList "Get user list"
// @Router /v1/users [get]
func GetList(c *gin.Context) {
	page := c.Query("page")
	limit := c.Query("limit")

	pageInt := 0
	limitInt := 0
	if page != "" {
		pageInt, _ = strconv.Atoi(page)
	}
	if limit != "" {
		limitInt, _ = strconv.Atoi(limit)
	}

	u, _ := service.Svc.UserSvc.GetUserPageable(
		context.TODO(),
		pageInt, limitInt,
	)
	api.SendResponse(c, nil, u)
}
