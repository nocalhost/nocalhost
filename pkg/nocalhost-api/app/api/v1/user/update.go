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

	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		api.SendResponse(c, errno.ErrUpdateUserDenied, nil)
		return
	}

	pwd := ""
	var err error
	if req.Password != "" {
		pwd, err = auth.Encrypt(req.Password)
		if err != nil {
			api.SendResponse(c, errno.InternalServerError, nil)
			return
		}
	}

	userMap := model.UserBaseModel{
		Email:    req.Email,
		Name:     req.Name,
		Password: pwd,
		Status:   req.Status,
	}
	//userMap := make(map[string]interface{})
	//userMap["email"] = req.Email
	//userMap["name"] = req.Name
	//userMap["password"] = pwd
	//userMap["status"] = req.Status
	result, err := service.Svc.UserSvc().UpdateUser(context.TODO(), userId, &userMap)
	if err != nil {
		log.Warnf("[user] update user err, %v", err)
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	api.SendResponse(c, nil, result)
}
