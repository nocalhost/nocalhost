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

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Update 更新用户信息
// @Summary 更新用户信息
// @Description Update a user by ID
// @Tags 用户
// @Accept  json
// @Produce  json
// @Param id path uint64 true "The user's database id index num"
// @Param user body model.UserBaseModel true "The user info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /users/{id} [put]
func Update(c *gin.Context) {
	// Get the user id from the url parameter.
	userID := cast.ToUint64(c.Param("id"))

	// Binding the user data.
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		log.Warnf("bind request param err: %+v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	log.Infof("user update req: %#v", req)

	userMap := make(map[string]interface{})
	userMap["avatar"] = req.Avatar
	userMap["sex"] = req.Sex
	err := service.Svc.UserSvc().UpdateUser(context.TODO(), uint64(userID), userMap)
	if err != nil {
		log.Warnf("[user] update user err, %v", err)
		api.SendResponse(c, errno.InternalServerError, nil)
		return
	}

	api.SendResponse(c, nil, userID)
}
