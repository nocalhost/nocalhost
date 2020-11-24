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

package cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
)

// GetList 获取集群列表
// @Summary 获取集群列表
// @Description 获取集群列表
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.ClusterList "{"code":0,"message":"OK","data":model.ClusterList}"
// @Router /v1/cluster [get]
func GetList(c *gin.Context) {
	result, _ := service.Svc.ClusterSvc().GetList(c)
	api.SendResponse(c, nil, result)
}

// @Summary 集群开发环境列表
// @Description 集群入口获取集群开发环境
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "集群 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig"
// @Router /v1/cluster/{id}/dev_space [get]
func GetSpaceList(c *gin.Context) {
	//userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	where := model.ClusterUserModel{
		ClusterId: clusterId,
	}
	result, err := service.Svc.ClusterUser().GetList(c, where)
	if err != nil {
		api.SendResponse(c, nil, make([]interface{}, 0))
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary 获取集群详情
// @Description 获取集群详情
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "集群 ID"
// @Success 200 {object} model.ClusterModel "应用开发环境参数，含 kubeconfig"
// @Router /v1/cluster/{id}/detail [get]
func GetDetail(c *gin.Context) {
	userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ClusterSvc().Get(c, clusterId, userId.(uint64))
	if err != nil {
		api.SendResponse(c, nil, make([]interface{}, 0))
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary 集群某个开发环境的详情
// @Description 通过集群 id 和开发环境 id 获取集群开发环境详情
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "集群 ID"
// @Param space_id path string true "开发空间 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig"
// @Router /v1/cluster/{id}/dev_space/{space_id}/detail [get]
func GetSpaceDetail(c *gin.Context) {
	//userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	devSpaceId := cast.ToUint64(c.Param("space_id"))
	where := model.ClusterUserModel{
		ID:        devSpaceId,
		ClusterId: clusterId,
	}
	result, err := service.Svc.ClusterUser().GetFirst(c, where)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
