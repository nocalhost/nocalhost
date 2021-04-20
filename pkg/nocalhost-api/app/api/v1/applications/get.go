/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package applications

import (
	"fmt"
	"github.com/spf13/cast"
	"nocalhost/cmd/nhctl/cmds/tpl"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
)

// Create Get Application
// @Summary Get Application
// @Description Get Application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":[{"id":1,"context":"application info","status":"1"}]}"
// @Router /v1/application [get]
func Get(c *gin.Context) {

	var params ApplicationListQuery

	err := c.ShouldBindQuery(&params)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	if ginbase.IsAdmin(c) {

		owner, err := listOwner(c, nil)
		if err != nil {
			api.SendResponse(c, errno.ErrApplicationGet, nil)
			return
		}

		api.SendResponse(c, errno.OK, owner)
	} else {

		user, err := ginbase.LoginUser(c)
		if err != nil {
			api.SendResponse(c, errno.ErrPermissionDenied, nil)
		}

		permitted, err := listPermitted(c, user)
		if err != nil {
			api.SendResponse(c, errno.ErrApplicationGet, nil)
			return
		}

		api.SendResponse(c, errno.OK, permitted)
	}
}

// list permitted applications (for normal user)
func ListPermitted(c *gin.Context) {
	userId := cast.ToUint64(c.Param("id"))

	permitted, err := listPermitted(c, userId)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}

	api.SendResponse(c, errno.OK, permitted)
}

// Create Get Application Detail
// @Summary Get Application Detail
// @Description Get Application Detail
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Application ID"
// @Success 200 {object} model.ApplicationModel
// @Router /v1/application/{id} [get]
func GetDetail(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	// userId, _ := c.Get("userId")
	result, err := service.Svc.ApplicationSvc().Get(c, applicationId)
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}

	currentUser, _ := ginbase.LoginUser(c)
	result.FillEditable(ginbase.IsAdmin(c), currentUser)

	api.SendResponse(c, errno.OK, result)
}

// @Summary Get authorized details of the application (obsolete)
// @Description Get authorized details of the application (obsolete)
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param clusterId path string true "Cluster ID"
// @Param id path string true "Application ID"
// @Success 200 {object} model.ClusterUserModel ""
// @Router /v1/application/{id}/cluster/{clusterId} [get]
func GetSpaceDetail(c *gin.Context) {
	// userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("clusterId"))
	applicationId := cast.ToUint64(c.Param("id"))
	models := model.ClusterUserModel{
		// UserId:        userId.(uint64),
		ClusterId: clusterId,
		ApplicationId: applicationId,
	}
	result, err := service.Svc.ClusterUser().GetList(c, models)
	if err != nil {
		api.SendResponse(c, nil, make([]interface{}, 0))
		return
	}
	api.SendResponse(c, nil, result)
}

// Create Plug-in access to applications (including installation status)
// @Summary Plug-in access to applications (including installation status)
// @Description Plug-in access to applications (including installation status)
// @Tags Plug-in
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.PluginApplicationModel
// @Router /v1/plugin/dev_space [get]
func PluginGet(c *gin.Context) {
	userId, _ := c.Get("userId")
	result, err := service.Svc.ApplicationSvc().PluginGetList(c, userId.(uint64))
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}
	// get plugin dev start append command
	for k := range result {
		result[k].DevStartAppendCommand = fmt.Sprintf(
			"%s %s", global.NocalhostDefaultPriorityclassKey, global.NocalhostDefaultPriorityclassName,
		)
	}
	api.SendResponse(c, errno.OK, result)
}

// GetNocalhostConfigTemplate get nocalhost config template
// @Summary get nocalhost config template
// @Description get nocalhost config template
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":{"template":""}}"
// @Router /v1/nocalhost/templates [get]
func GetNocalhostConfigTemplate(c *gin.Context) {
	result := map[string]string{
		"template": tpl.CombineTpl(),
	}
	api.SendResponse(c, errno.OK, result)
}

// list owner only list the application user created
func listOwner(c *gin.Context, userId *uint64) ([]*model.ApplicationModel, error) {
	result, err := service.Svc.ApplicationSvc().GetList(c, userId)
	if err != nil {
		log.Warnf("get Application err: %v", err)
		return nil, err
	}

	for _, applicationModel := range result {
		currentUser, _ := ginbase.LoginUser(c)
		applicationModel.FillEditable(ginbase.IsAdmin(c), currentUser)
	}

	return result, nil
}

// list permitted is list all application user has
func listPermitted(c *gin.Context, userId uint64) ([]*model.ApplicationModel, error) {
	// permitted
	applicationUsers, err := service.Svc.ApplicationUser().ListByUserId(c, userId)
	if err != nil {
		log.Warnf("get application_user err: %v", err)
		return nil, err
	}

	set := map[uint64]interface{}{}
	for _, au := range applicationUsers {
		set[au.ApplicationId] = "-"
	}

	// userId, _ := c.Get("userId")
	lists, err := service.Svc.ApplicationSvc().GetList(c, nil)
	if err != nil {
		log.Warnf("get Application err: %v", err)
		return nil, err
	}

	var result []*model.ApplicationModel
	for _, app := range lists {
		_, ok := set[app.ID]

		// public
		if app.Public == 1 ||

			// creator
			app.UserId == userId ||

			// has permission
			ok {

			currentUser, _ := ginbase.LoginUser(c)
			app.FillEditable(ginbase.IsAdmin(c), currentUser)
			result = append(result, app)
		}
	}

	return result, nil
}
