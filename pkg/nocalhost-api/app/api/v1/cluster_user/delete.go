/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

// Delete Completely delete the development environment
// @Summary Completely delete the development environment
// @Description Completely delete the development environment, including deleting the K8S namespace
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/dev_space/{id} [delete]
func Delete(c *gin.Context) {

	devSpaceId := cast.ToUint64(c.Param("id"))
	clusterUser, errn := HasHighPermissionToSomeDevSpace(c, devSpaceId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	if clusterUser.IsClusterAdmin() {

		if err := cluster_scope.RemoveAllFromViewer(clusterUser.ClusterId, clusterUser.UserId); err != nil {
			api.SendResponse(c, err, nil)
			return
		}

		if err := cluster_scope.RemoveAllFromCooperator(clusterUser.ClusterId, clusterUser.UserId); err != nil {
			api.SendResponse(c, err, nil)
			return
		}

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
	}

	clusterData, err := service.Svc.ClusterSvc().Get(c, clusterUser.ClusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	meshDevInfo := &setupcluster.MeshDevInfo{
		Header: clusterUser.TraceHeader,
	}
	req := ClusterUserCreateRequest{
		ID:             &clusterUser.ID,
		NameSpace:      clusterUser.Namespace,
		BaseDevSpaceId: clusterUser.BaseDevSpaceId,
		MeshDevInfo:    meshDevInfo,
	}
	devSpace := NewDevSpace(req, c, []byte(clusterData.KubeConfig))

	if err := devSpace.Delete(); err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	devSpaces, err := service.Svc.ClusterUser().GetList(c, model.ClusterUserModel{BaseDevSpaceId: devSpaceId})
	if err != nil {
		// can not find mesh dev space, do nothing
		api.SendResponse(c, errno.OK, nil)
		return
	}

	// delete the dev spaces that the bash space is the one we want to delete
	for _, space := range devSpaces {
		clusterData, err := service.Svc.ClusterSvc().Get(c, space.ClusterId)
		if err != nil {
			api.SendResponse(c, errno.ErrClusterNotFound, nil)
			return
		}
		meshDevInfo := &setupcluster.MeshDevInfo{
			Header: space.TraceHeader,
		}
		req := ClusterUserCreateRequest{
			ID:             &space.ID,
			NameSpace:      space.Namespace,
			BaseDevSpaceId: space.BaseDevSpaceId,
			MeshDevInfo:    meshDevInfo,
		}
		devSpace := NewDevSpace(req, c, []byte(clusterData.KubeConfig))
		if err := devSpace.Delete(); err != nil {
			api.SendResponse(c, err, nil)
			return
		}
	}

	api.SendResponse(c, errno.OK, nil)

}

// ReCreate ReCreate devSpace
// @Summary ReCreate devSpace
// @Description delete devSpace and create a new one
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/dev_space/{id}/recreate [post]
func ReCreate(c *gin.Context) {
	// get devSpace
	devSpaceId := cast.ToUint64(c.Param("id"))

	user, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	clusterUser, errn := HasModifyPermissionToSomeDevSpace(c, devSpaceId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	res := SpaceResourceLimit{}
	json.Unmarshal([]byte(clusterUser.SpaceResourceLimit), &res)
	// create a new dev space
	meshDevInfo := &setupcluster.MeshDevInfo{
		Header: clusterUser.TraceHeader,
	}
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
		BaseDevSpaceId:     clusterUser.BaseDevSpaceId,
		MeshDevInfo:        meshDevInfo,
	}

	cluster, err := service.Svc.ClusterSvc().GetCache(clusterUser.ClusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	// delete devSpace space first, it will delete database record whatever success delete namespace or not
	devSpace := NewDevSpace(req, c, []byte(cluster.KubeConfig))

	list, e := DoList(&model.ClusterUserModel{ID: devSpaceId}, user, false)
	if e != nil {
		log.ErrorE(e, "")
		api.SendResponse(c, err, nil)
	}
	if len(list) != 1 {
		api.SendResponse(c, errno.ErrMeshClusterUserNotFound, nil)
	}
	cu := list[0]

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

	for _, viewer := range cu.ViewerUser {
		_ = ns_scope.AsViewer(result.ClusterId, viewer.ID, result.Namespace)
	}

	for _, cooper := range cu.CooperUser {
		_ = ns_scope.AsCooperator(result.ClusterId, cooper.ID, result.Namespace)
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
