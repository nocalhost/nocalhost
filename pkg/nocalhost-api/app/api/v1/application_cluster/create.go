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

package application_cluster

import (
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// Create 关联集群
// @Summary 关联集群
// @Description 应用关联集群
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body application_cluster.ApplicationClusterRequest true "The application info"
// @Param id path uint64 true "应用 ID"
// @Success 200 {object} model.ApplicationClusterModel
// @Router /v1/application/{id}/bind_cluster [post]
func Create(c *gin.Context) {
	var req ApplicationClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind ApplicationCluster params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	// check application auth
	if _, err := service.Svc.ApplicationSvc().Get(c, applicationId, userId.(uint64)); err != nil {
		api.SendResponse(c, errno.ErrPermissionApplication, nil)
		return
	}
	// check cluster auth
	if _, err := service.Svc.ClusterSvc().Get(c, *req.ClusterId, userId.(uint64)); err != nil {
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}
	result, err := service.Svc.ApplicationClusterSvc().Create(c, applicationId, *req.ClusterId, userId.(uint64))
	if err != nil {
		log.Warnf("create ApplicationCluster err: %v", err)
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}

	api.SendResponse(c, nil, result)
}
