/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cluster_user

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// Delete Completely delete the development environment
// @Summary Completely delete the development environment
// @Description Completely delete the development environment, including deleting the K8S namespace
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/dev_space/{id} [delete]
func Delete(c *gin.Context) {

	devSpaceId := cast.ToUint64(c.Param("id"))
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	if clusterUser.ClusterAdmin != nil && *clusterUser.ClusterAdmin != 0 {

		if err := service.Svc.UnAuthorizeClusterToUser(clusterUser.ClusterId, clusterUser.UserId); err != nil {
			api.SendResponse(c, err, nil)
			return
		}

		// delete database cluster-user dev space
		if err := service.Svc.ClusterUser().Delete(c, devSpaceId); err != nil {
			api.SendResponse(c, errno.ErrDeletedClusterButDatabaseFail, nil)
			return
		}

		api.SendResponse(c, errno.OK, nil)
		return
	} else {
		clusterData, err := service.Svc.ClusterSvc().Get(c, clusterUser.ClusterId)
		if err != nil {
			api.SendResponse(c, errno.ErrClusterNotFound, nil)
			return
		}

		req := ClusterUserCreateRequest{
			ID:        &clusterUser.ID,
			NameSpace: clusterUser.Namespace,
		}
		devSpace := NewDevSpace(req, c, []byte(clusterData.KubeConfig))

		if err := devSpace.Delete(); err != nil {
			api.SendResponse(c, err, nil)
			return
		}

		api.SendResponse(c, errno.OK, nil)
	}
}

// ReCreate ReCreate devSpace
// @Summary ReCreate devSpace
// @Description delete devSpace and create a new one
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/dev_space/{id}/recreate [post]
func ReCreate(c *gin.Context) {
	// get devSpace
	devSpaceId := cast.ToUint64(c.Param("id"))
	condition := model.ClusterUserModel{
		ID: devSpaceId,
	}
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		userId, _ := c.Get("userId")
		condition.UserId = cast.ToUint64(userId)
	}
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, condition)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	// refuse to recreate cluster_admin devSpace
	if clusterUser.ClusterAdmin != nil && *clusterUser.ClusterAdmin != uint64(0) {
		api.SendResponse(c, nil, nil)
		return
	}

	clusterData, err := service.Svc.ClusterSvc().Get(c, clusterUser.ClusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	res := SpaceResourceLimit{}
	json.Unmarshal([]byte(clusterUser.SpaceResourceLimit), &res)
	// create a new dev space
	req := ClusterUserCreateRequest{
		ClusterId:          &clusterUser.ClusterId,
		UserId:             &clusterUser.UserId,
		SpaceName:          clusterUser.SpaceName,
		Memory:             &clusterUser.Memory,
		Cpu:                &clusterUser.Cpu,
		ApplicationId:      &clusterUser.ApplicationId,
		NameSpace:          clusterUser.Namespace,
		ID:                 &clusterUser.ID,
		SpaceResourceLimit: &res,
	}

	// delete devSpace space first, it will delete database record whatever success delete namespace or not
	devSpace := NewDevSpace(req, c, []byte(clusterData.KubeConfig))
	err = devSpace.Delete()
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	result, err := devSpace.Create()

	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	// un authorize namespace to user
	if err := service.Svc.AuthorizeNsToUser(result.ClusterId, result.UserId, result.Namespace); err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	api.SendResponse(c, nil, result)
}

// ReCreate Plugin ReCreate devSpace
// @Summary Plugin ReCreate devSpace
// @Description Plugin delete devSpace and create a new one
// @Tags Plug-in
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/plugin/{id}/recreate [post]
func PluginReCreate(c *gin.Context) {
	// check permission
	userId, _ := c.Get("userId")
	devSpaceId := cast.ToUint64(c.Param("id"))
	_, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId, UserId: userId.(uint64)})
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	ReCreate(c)
}
