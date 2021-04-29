/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package user

import (
	"context"
	"github.com/spf13/cast"

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
	u, err := service.Svc.UserSvc().GetUserByID(context.TODO(), userID)
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
	u, err := service.Svc.UserSvc().GetUserByID(context.TODO(), userID.(uint64))
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
	u, _ := service.Svc.UserSvc().GetUserList(context.TODO())
	api.SendResponse(c, nil, u)
}
