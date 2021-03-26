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
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func PublicSwitch(c *gin.Context) {
	var req AppPublicSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("update application bind err: %s", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	if !ginbase.IsAdmin(c) {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
	}

	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	err := service.Svc.ApplicationSvc().PublicSwitch(c, applicationId, *req.Public)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}

// Create Edit application
// @Summary Edit application
// @Description Edit application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "Application ID"
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

	// adapt earlier version
	if req.Public == nil {
		u := uint8(1)
		req.Public = &u
	}

	// normal user can't not create public applications
	if !ginbase.IsAdmin(c) {
		deny := uint8(0)
		req.Public = &deny
	}

	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	model := model.ApplicationModel{
		ID: applicationId,
		// UserId:  userId.(uint64),
		Context: req.Context,
		Status:  *req.Status,
		Public:  *req.Public,
	}
	result, err := service.Svc.ApplicationSvc().Update(c, &model)
	if err != nil {
		log.Warnf("update Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationUpdate, nil)
		return
	}

	api.SendResponse(c, errno.OK, result)
}

// Create Plug-in Update app installation status
// @Summary Plug-in Update app installation status
// @Description Plug-in Update app installation status
// @Tags Plug-in
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "Application ID"
// @Param spaceId path uint64 true "DevSpace ID"
// @Param CreateAppRequest body applications.UpdateApplicationInstallRequest true "The application update info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/plugin/application/{id}/dev_space/{spaceId}/plugin_sync [put]
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
