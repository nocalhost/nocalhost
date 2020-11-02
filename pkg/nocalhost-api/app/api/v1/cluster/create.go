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
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/gin-gonic/gin"
)

// Create 添加集群
// @Summary 添加集群
// @Description 用户添加集群，暂时不考虑验证集群的 kubeconfig
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param createCluster body cluster.CreateClusterRequest true "The cluster info"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/cluster [post]
func Create(c *gin.Context) {
	var req CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("createCluster bind params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	err := service.Svc.ClusterSvc().Create(c, req.Name, req.Marks, req.KubeConfig, userId.(uint64))
	if err != nil {
		log.Warnf("create cluster err: %v", err)
		api.SendResponse(c, errno.ErrClusterCreate, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
