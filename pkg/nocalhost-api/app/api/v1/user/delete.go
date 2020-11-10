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
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// Create 删除用户
// @Summary 删除用户
// @Description 管理员删除用户
// @Tags 用户
// @Accept json
// @Produce json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "用户 ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/users/{id} [delete]
func Delete(c *gin.Context) {
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		api.SendResponse(c, errno.ErrUpdateUserDenied, nil)
		return
	}
	userId := cast.ToUint64(c.Param("id"))
	err := service.Svc.UserSvc().Delete(c, userId)
	if err != nil {
		log.Warnf("user delete error: %v", err)
		api.SendResponse(c, errno.ErrDeleteUser, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
