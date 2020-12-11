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
	"strings"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Login Web and plug-in login
// @Summary Web and plug-in login
// @Description Web and plug-in login
// @Tags Users
// @Produce  json
// @Param login body user.LoginCredentials true "Login user info"
// @Success 200 {string} json "{"code":0,"message":"OK","data":{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"}}"
// @Router /v1/login [post]
func Login(c *gin.Context) {
	// Binding the data with the u struct.
	var req LoginCredentials
	if err := c.Bind(&req); err != nil {
		log.Warnf("email login bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, err)
		return
	}

	log.Infof("login req %#v", req)
	// check param
	if req.Email == "" || req.Password == "" {
		log.Warnf("email or password is empty: %v", req)
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}
	// By default, “From” is not passed in web login, and ordinary users are prohibited from logging in to the web interface
	users, err := service.Svc.UserSvc().GetUserByEmail(c, req.Email)
	if err != nil {
		api.SendResponse(c, errno.ErrEmailOrPassword, nil)
		return
	}

	// Log in to the web
	if users.IsAdmin != 1 && req.From != "plugin" {
		api.SendResponse(c, errno.ErrUserLoginWebNotAllow, nil)
		return
	}

	t, err := service.Svc.UserSvc().EmailLogin(c, req.Email, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "allow") {
			api.SendResponse(c, errno.ErrUserNotAllow, nil)
			return
		}
		log.Warnf("email login err: %v", err)
		api.SendResponse(c, errno.ErrEmailOrPassword, nil)
		return
	}

	api.SendResponse(c, nil, model.Token{
		Token: t,
	})
}
