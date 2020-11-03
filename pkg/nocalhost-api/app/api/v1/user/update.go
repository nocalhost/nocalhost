/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package user

import (
	"context"
	"nocalhost/pkg/nocalhost-api/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Update 更新用户信息
// @Summary 更新用户信息（含禁用用户）
// @Description Update a user by ID
// @Tags 用户
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "The user's database id index num"
// @Param user body user.CreateUserRequest true "Update user info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/users/{id} [put]
func Update(c *gin.Context) {
	// Get the user id from the url parameter.
	userId := cast.ToUint64(c.Param("id"))

	// Binding the user data.
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		log.Warnf("bind request param err: %+v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	if req.Password != req.ConfirmPassword {
		log.Warnf("twice password is not same")
		api.SendResponse(c, errno.ErrTwicePasswordNotMatch, nil)
		return
	}

	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		api.SendResponse(c, errno.ErrUpdateUserDenied, nil)
		return
	}

	pwd, err := auth.Encrypt(req.Password)
	if err != nil {
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	userMap := make(map[string]interface{})
	userMap["email"] = req.Email
	userMap["name"] = req.Name
	userMap["password"] = pwd
	userMap["status"] = req.Status
	err = service.Svc.UserSvc().UpdateUser(context.TODO(), userId, &userMap)
	if err != nil {
		log.Warnf("[user] update user err, %v", err)
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
