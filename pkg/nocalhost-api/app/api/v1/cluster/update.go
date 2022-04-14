/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/middleware"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// @Summary Update cluster
// @Description Update cluster
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Cluster ID"
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
	updateCol := map[string]interface{}{
		"storage_class": req.StorageClass,
	}
	cluster, err2 := service.Svc.ClusterSvc.Get(c, clusterId)
	if err2 != nil {
		api.SendResponse(c, errno.ErrUpdateCluster, nil)
		return
	}
	if admin, _ := middleware.IsAdmin(c); !admin && cluster.UserId != c.GetUint64("userId") {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}
	result, err := service.Svc.ClusterSvc.Update(c, updateCol, clusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrUpdateCluster, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
