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
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// Create Delete Application
// @Summary Delete Application
// @Description The user deletes the application, and also deletes the configured development space in the application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id} [delete]
func Delete(c *gin.Context) {
	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))

	if !ginbase.IsAdmin(c) {
		get, err := service.Svc.ApplicationSvc().Get(c, applicationId)
		if err != nil {
			api.SendResponse(c, errno.ErrApplicationDelete, nil)
			return
		}

		if ginbase.IsCurrentUser(c, get.UserId) {
			api.SendResponse(c, errno.ErrPermissionDenied, nil)
			return
		}
	}

	// delete application database record
	err := service.Svc.ApplicationSvc().Delete(c, applicationId)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationDelete, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
