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
	clusterUser, errn := HasPrivilegeToSomeDevSpace(c, devSpaceId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	if clusterUser.Protected {
		api.SendResponse(c, errno.ErrProtectedSpaceReSet, nil)
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
		if err := service.Svc.ClusterUserSvc.Delete(c, devSpaceId); err != nil {
			api.SendResponse(c, errno.ErrDeletedClusterButDatabaseFail, nil)
			return
		}

		api.SendResponse(c, errno.OK, nil)
		return
	}

	clusterData, err := service.Svc.ClusterSvc.Get(c, clusterUser.ClusterId)
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
		IsBaseSpace:    clusterUser.IsBaseSpace,
	}
	devSpace := NewDevSpace(req, c, []byte(clusterData.KubeConfig))

	if err := devSpace.Delete(); err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	// delete share space when deleting base space
	if clusterUser.IsBaseSpace {
		deleteShareSpaces(c, devSpaceId)
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

	clusterUser, errn := LoginUserHasModifyPermissionToSomeDevSpace(c, devSpaceId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	if clusterUser.Protected {
		api.SendResponse(c, errno.ErrProtectedSpaceReSet, nil)
		return
	}

	// base space can't be reset
	if clusterUser.IsBaseSpace {
		api.SendResponse(c, errno.ErrBaseSpaceReSet, nil)
		return
	}

	res := SpaceResourceLimit{}
	_ = json.Unmarshal([]byte(clusterUser.SpaceResourceLimit), &res)
	// create a new dev space
	meshDevInfo := &setupcluster.MeshDevInfo{
		Header:   clusterUser.TraceHeader,
		ReCreate: true,
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
		IsBaseSpace:        clusterUser.IsBaseSpace,
	}

	cluster, err := service.Svc.ClusterSvc.GetCache(clusterUser.ClusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	// delete devSpace space first, it will delete database record whatever success delete namespace or not
	devSpace := NewDevSpace(req, c, []byte(cluster.KubeConfig))

	list, e := DoList(&model.ClusterUserModel{ID: devSpaceId}, user, ginbase.IsAdmin(c), false)
	if e != nil {
		log.ErrorE(e, "")
		api.SendResponse(c, err, nil)
		return
	}
	if len(list) != 1 {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	cu := list[0]

	if cu.IsClusterAdmin() {
		api.SendResponse(c, nil, nil)
		return
	}

	err = devSpace.Delete()
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}

	// set namespace to empty, to recreate a namespace
	devSpace.DevSpaceParams.NameSpace = ""
	result, err := devSpace.Create()

	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	if err := service.Svc.AuthorizeNsToUser(result.ClusterId, result.UserId, result.Namespace); err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	if err := service.Svc.AuthorizeNsToDefaultSa(result.ClusterId, result.UserId, result.Namespace); err != nil {
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
	//userId, _ := c.Get("userId")
	devSpaceId := cast.ToUint64(c.Param("id"))
	_, err := service.Svc.ClusterUserSvc.GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	ReCreate(c)
}
