/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/middleware"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

// GetList Delete the cluster completely
// @Summary Delete the cluster completely
// @Description Delete the cluster completely
// @Tags Cluster
// @Accept  json
// @Produce  json
// @Param Authorization header string true "Authorization"
// @Param id path uint64 true "Cluster ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/cluster/{id} [delete]
func Delete(c *gin.Context) {
	userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	cluster, err := service.Svc.ClusterSvc.Get(c, clusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}
	// admin can delete all cluster, but normal user can delete cluster they created only
	if isAdmin, _ := middleware.IsAdmin(c); !isAdmin && (userId != cluster.UserId) {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	deleteMeshManager(cluster.KubeConfig)

	goClient, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		log.Warnf("Cannot connect to this kubernetes cluster, err %s", err.Error())
	}

	// get all dev space
	condition := model.ClusterUserModel{
		ClusterId: clusterId,
	}
	devSpace, err := service.Svc.ClusterUserSvc.GetList(c, condition)
	var spaceIds []uint64
	var spaceNames []string
	if len(devSpace) > 0 {
		for _, space := range devSpace {
			spaceIds = append(spaceIds, space.ID)
			spaceNames = append(spaceNames, space.Namespace)
		}
	}
	releaseTargetClusterResources(goClient, clusterId, spaceNames)
	result := deleteNocalhostManagedData(c, clusterId, spaceIds)
	if !result {
		return
	}
	Remove(cluster.KubeConfig)
	api.SendResponse(c, errno.OK, nil)
}

// Release target kubernetes cluster resources.
func releaseTargetClusterResources(goClient *clientgo.GoClient, clusterId uint64, spaceNames []string) {
	if goClient != nil {
		_, err := goClient.DeleteNS(global.NocalhostSystemNamespace)
		if err != nil {
			// ignore fail
			log.Warnf("delete cluster for id %s fail, err %s", clusterId, err.Error())
		}
		for _, spaceName := range spaceNames {
			_, err := goClient.DeleteNS(spaceName)
			if err != nil {
				log.Warnf("delete devspace for spaceName %s fail, err %s", spaceName, err.Error())
			}
		}
	}
}

// Delete cluster data managed by nocalhost. such as: cluster and cluster users
func deleteNocalhostManagedData(c *gin.Context, clusterId uint64, spaceIds []uint64) bool {
	err := service.Svc.ClusterSvc.Delete(c, clusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrDeletedClusterDBButClusterDone, nil)
		return false
	}

	if len(spaceIds) > 0 {
		err = service.Svc.ClusterUserSvc.BatchDelete(c, spaceIds)
		if err != nil {
			api.SendResponse(c, errno.ErrDeletedClusterDevSpaceDBButClusterDone, nil)
			return false
		}
	}
	return true
}

// Delete manager form factory
func deleteMeshManager(kubeconfig string) {
	setupcluster.GetSharedMeshManagerFactory().Delete(kubeconfig)
}
