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
	"github.com/spf13/cast"
	"nocalhost/cmd/nhctl/cmds/tpl"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
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
	// userId, _ := c.Get("userId")
	result, err := service.Svc.ApplicationSvc().GetList(c)
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}

	api.SendResponse(c, errno.OK, result)
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
		ClusterId:     clusterId,
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
// @Router /v1/plugin/applications [get]
func PluginGet(c *gin.Context) {
	userId, _ := c.Get("userId")
	result, err := service.Svc.ApplicationSvc().PluginGetList(c, userId.(uint64))
	if err != nil {
		log.Warnf("get Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationGet, nil)
		return
	}
	// get plugin dev start append command
	// TODO
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
		"template": tpl.ConbineTpl(),
	}
	api.SendResponse(c, errno.OK, result)
}
