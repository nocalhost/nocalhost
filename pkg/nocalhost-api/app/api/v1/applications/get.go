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

package applications

import (
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
)

// Create 获取应用
// @Summary 获取应用
// @Description 用户获取应用
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":[{"id":1,"context":"application info","status":"1"}]}"
// @Router /v1/application [get]
func Get(c *gin.Context) {
	userId, _ := c.Get("userId")
	result, err := service.Svc.ApplicationSvc().GetList(c, userId.(uint64))
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}

	api.SendResponse(c, errno.OK, result)
}
