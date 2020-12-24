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
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// @Summary Update cluster
// @Description Update cluster
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Cluster ID"
// @Param space_id path string true "DevSpace ID"
// @Param createCluster body cluster.UpdateClusterRequest true "The cluster info"
// @Success 200 {object} model.ClusterModel "include kubeconfig"
// @Router /v1/cluster/{id} [put]
func Update(c *gin.Context) {
	var req UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	clusterId := cast.ToUint64(c.Param("id"))
	update := &model.ClusterModel{
		StorageClass: req.StorageClass,
	}
	err := service.Svc.ClusterSvc().Update(c, update, clusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrUpdateCluster, nil)
		return
	}
	api.SendResponse(c, nil, update)
}
