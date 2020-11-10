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
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Register 添加开发者
// @Summary 添加开发者
// @Description 管理员添加开发者
// @Tags 用户
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param register body user.CreateUserRequest true "Reg user info"
// @Success 200 {string} json "{"code":0,"message":"OK","data":null}"
// @Router /v1/users [post]
func Create(c *gin.Context) {
	// Binding the data with the u struct.
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("create user bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	// 两次密码是否正确
	if req.Password != req.ConfirmPassword {
		log.Warnf("twice password is not same")
		api.SendResponse(c, errno.ErrTwicePasswordNotMatch, nil)
		return
	}

	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		api.SendResponse(c, errno.ErrCreateUserDenied, nil)
		return
	}

	err := service.Svc.UserSvc().Create(c, req.Email, req.Password, req.Name, *req.Status)
	if err != nil {
		log.Warnf("register err: %v", err)
		api.SendResponse(c, errno.ErrRegisterFailed, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
