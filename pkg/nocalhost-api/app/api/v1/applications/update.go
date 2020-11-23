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
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// Create 编辑应用
// @Summary 编辑应用
// @Description 用户编辑应用
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "应用 ID"
// @Param CreateAppRequest body applications.CreateAppRequest true "The application info"
// @Success 200 {object} model.ApplicationModel
// @Router /v1/application/{id} [put]
func Update(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("update application bind err: %s", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	model := model.ApplicationModel{
		ID:      applicationId,
		UserId:  userId.(uint64),
		Context: req.Context,
		Status:  *req.Status,
	}
	result, err := service.Svc.ApplicationSvc().Update(c, &model)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, result)
}

// Create Plugin - 更新应用安装状态
// @Summary Plugin - 更新应用安装状态
// @Description Plugin - 更新应用安装状态
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "应用 ID"
// @Param spaceId path uint64 true "开发空间 ID"
// @Param CreateAppRequest body applications.UpdateApplicationInstallRequest true "The application update info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id}/dev_space/{spaceId}/plugin_sync [put]
func UpdateApplicationInstall(c *gin.Context) {
	var req UpdateApplicationInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("update application bind err: %s", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	devSpaceId := cast.ToUint64(c.Param("spaceId"))
	model := model.ClusterUserModel{
		ID:            devSpaceId,
		ApplicationId: applicationId,
		UserId:        userId.(uint64),
		Status:        req.Status,
	}
	_, err := service.Svc.ClusterUser().Update(c, &model)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationInstallUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
