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

package cluster_user

import (
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"

	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
)

// @Summary Plugin - 获取个人应用开发环境(kubeconfig)
// @Description Get user's application dev space
// @Tags 插件
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "应用 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig，status=0应用未安装，1已安装"
// @Router /v1/application/{id}/dev_space [get]
func GetFirst(c *gin.Context) {
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId.(uint64),
	}
	result, err := service.Svc.ClusterUser().GetFirst(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary 获取应用开发环境列表
// @Description Get application dev space list
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "应用 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig，status=0 应用未安装，1已安装"
// @Router /v1/application/{id}/dev_space_list [get]
func GetList(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
	}
	result, err := service.Svc.ClusterUser().GetList(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary 获取应用某个开发环境详情
// @Description Get dev space detail from application
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "应用 ID"
// @Param space_id path string true "开发空间 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig，status=0应用未安装，1已安装"
// @Router /v1/application/{id}/dev_space/{space_id}/detail [get]
func GetDevSpaceDetail(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	spaceId := cast.ToUint64(c.Param("space_id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		ID:            spaceId,
	}
	result, err := service.Svc.ClusterUser().GetFirst(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
