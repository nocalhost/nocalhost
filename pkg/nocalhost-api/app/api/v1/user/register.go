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

// Register Registration(obsolete)
// @Summary The administrator adds users, users cannot register themselves
// @Description Registration(obsolete)
// @Tags Users
// @Produce  json
// @Param register body user.RegisterRequest true "Reg user info"
// @Success 200 {string} json "{"code":0,"message":"OK","data":{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"}}"
// @Router /v1/register [post]
func Register(c *gin.Context) {
	// Binding the data with the u struct.
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("register bind param err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	log.Infof("register req: %#v", req)
	// check param
	if req.Email == "" || req.Password == "" {
		log.Warnf("params is empty: %v", req)
		api.SendResponse(c, errno.ErrParam, nil)
		return
	}

	if req.Password != req.ConfirmPassword {
		log.Warnf("twice password is not same")
		api.SendResponse(c, errno.ErrTwicePasswordNotMatch, nil)
		return
	}

	err := service.Svc.UserSvc().Register(c, req.Email, req.Password)
	if err != nil {
		log.Warnf("register err: %v", err)
		api.SendResponse(c, errno.ErrRegisterFailed, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
