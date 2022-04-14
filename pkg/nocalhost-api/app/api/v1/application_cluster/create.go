/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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

// Create Associated cluster
// @Summary Associated cluster
// @Description Application associated cluster
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body application_cluster.ApplicationClusterRequest true "The application info"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} model.ApplicationClusterModel
// @Router /v1/application/{id}/bind_cluster [post]
func Create(c *gin.Context) {
	var req ApplicationClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind ApplicationCluster params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	// userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	// check application auth
	if _, err := service.Svc.ApplicationSvc.Get(c, applicationId); err != nil {
		api.SendResponse(c, errno.ErrPermissionApplication, nil)
		return
	}
	// check cluster auth
	if _, err := service.Svc.ClusterSvc.Get(c, *req.ClusterId); err != nil {
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}
	result, err := service.Svc.ApplicationClusterSvc.Create(c, applicationId, *req.ClusterId)
	if err != nil {
		log.Warnf("create ApplicationCluster err: %v", err)
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}

	api.SendResponse(c, nil, result)
}
