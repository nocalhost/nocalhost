/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"encoding/base64"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

type DevSpaceRequest struct {
	KubeConfig string `json:"kubeconfig"`
	SpaceName  string `json:"space_name"`
}

// Update
// @Summary Update dev space
// @Description Update dev space
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "devspace id"
// @Param CreateAppRequest body cluster_user.DevSpaceRequest true "kubeconfig"
// @Success 200 {object} model.ClusterUserModel
// @Router /v1/dev_space/{id} [put]
func Update(c *gin.Context) {
	var req DevSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind dev space params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	devSpaceId := cast.ToUint64(c.Param("id"))
	sDec, err := base64.StdEncoding.DecodeString(req.KubeConfig)
	if err != nil {
		log.Warnf("bind dev space params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	cu := model.ClusterUserModel{
		ID:         devSpaceId,
		KubeConfig: string(sDec),
		SpaceName:  req.SpaceName,
	}
	result, err := service.Svc.ClusterUserSvc.Update(c, &cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// UpdateResourceLimit
// @Summary UpdateResourceLimit
// @Description update resource limit in dev space
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "devspace id"
// @Param SpaceResourceLimit body cluster_user.SpaceResourceLimit true "kubeconfig"
// @Success 200 {object} model.ClusterUserModel
// @Router /v1/dev_space/{id}/update_resource_limit [put]
func UpdateResourceLimit(c *gin.Context) {

	var req SpaceResourceLimit
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind resource limits params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	devSpaceId := cast.ToUint64(c.Param("id"))

	_, errn := HasPrivilegeToSomeDevSpace(c, devSpaceId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	// Validate DevSpace Resource limit parameter format.
	flag, message := ValidSpaceResourceLimit(req)
	if !flag {
		log.Errorf(
			"update devspace resource limit fail. "+
				"Incorrect resource limit parameter  [ %v ] format.", message,
		)
		api.SendResponse(c, errno.ErrFormatResourceLimitParam, message)
		return
	}
	condition := model.ClusterUserModel{
		ID: devSpaceId,
	}
	devspace, err := service.Svc.ClusterUserSvc.GetFirst(c, condition)
	if err != nil || devspace == nil {
		log.Errorf("Dev space has not found")
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	// Build goclient with administrator kubeconfig
	clusterData, err := service.Svc.ClusterSvc.Get(c, devspace.ClusterId)
	if err != nil {
		log.Errorf("Getting cluster information failed, cluster id = [ %v ] ", devspace.ClusterId)
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewAdminGoClient(KubeConfig)

	// get client go and check if is admin Kubeconfig
	if err != nil {
		switch err.(type) {
		case *errno.Errno:
			api.SendResponse(c, err, nil)
		default:
			api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		}
		return
	}
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)

	if !req.Validate() {
		api.SendResponse(c, errno.ErrValidateResourceQuota, nil)
		return
	}

	// Recreate ResourceQuota
	resourceQuotaName := "rq-" + devspace.Namespace
	clusterDevsSetUp.DeleteResourceQuota(resourceQuotaName, devspace.Namespace).CreateResourceQuota(
		resourceQuotaName, devspace.Namespace, req.SpaceReqMem,
		req.SpaceReqCpu, req.SpaceLimitsMem, req.SpaceLimitsCpu, req.SpaceStorageCapacity, req.SpaceEphemeralStorage,
		req.SpacePvcCount, req.SpaceLbCount,
	)

	// Recreate LimitRange
	limiRangeName := "lr-" + devspace.Namespace
	clusterDevsSetUp.DeleteLimitRange(limiRangeName, devspace.Namespace).CreateLimitRange(
		limiRangeName, devspace.Namespace,
		req.ContainerReqMem, req.ContainerLimitsMem, req.ContainerReqCpu, req.ContainerLimitsCpu,
		req.ContainerEphemeralStorage,
	)

	// Update database clustUser's spaceResourceLimit
	resSting, _ := json.Marshal(req)
	devspace.SpaceResourceLimit = string(resSting)
	result, err := service.Svc.ClusterUserSvc.Update(c, devspace)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// UpdateMeshDevSpaceInfo update mesh dev space info
// @Summary Update mesh dev space info
// @Description Update mesh dev space info
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "devspace id"
// @Param MeshDevInfo body setupcluster.MeshDevInfo true "mesh dev space info"
// @Success 200 {object} model.ClusterUserModel
// @Router /v1/dev_space/{id}/update_mesh_dev_space_info [put]
func UpdateMeshDevSpaceInfo(c *gin.Context) {
	var req setupcluster.MeshDevInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind resource limits params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	devSpaceId := cast.ToUint64(c.Param("id"))

	condition := model.ClusterUserModel{
		ID: devSpaceId,
	}
	devspace, err := service.Svc.ClusterUserSvc.GetFirst(c, condition)
	if err != nil || devspace == nil {
		log.Errorf("Dev space has not found")
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	if devspace.BaseDevSpaceId == 0 {
		log.Errorf("Base space can't be updated")
		api.SendResponse(c, errno.ErrUpdateBaseSpace, nil)
		return
	}

	// check base dev space
	baseCondition := model.ClusterUserModel{
		ID: devspace.BaseDevSpaceId,
	}
	basespace, err := service.Svc.ClusterUserSvc.GetFirst(c, baseCondition)
	if err != nil || basespace == nil {
		log.Errorf("Base space has not found")
		api.SendResponse(c, errno.ErrMeshClusterUserNotFound, nil)
		return
	}

	// Build goclient with administrator kubeconfig
	clusterData, err := service.Svc.ClusterSvc.Get(c, devspace.ClusterId)
	if err != nil {
		log.Errorf("Getting cluster information failed, cluster id = [ %v ] ", devspace.ClusterId)
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}

	info := req
	info.MeshDevNamespace = devspace.Namespace
	info.BaseNamespace = basespace.Namespace
	info.IsUpdateHeader = basespace.TraceHeader.TraceKey != info.Header.TraceKey ||
		basespace.TraceHeader.TraceValue != info.Header.TraceValue

	meshManager, err := setupcluster.GetSharedMeshManagerFactory().Manager(clusterData.KubeConfig)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrUpdateMeshSpaceFailed, nil)
		return
	}

	log.Debugf("update mesh info for dev space %s, the namespace is %s", devspace.SpaceName, devspace.Namespace)
	if err := meshManager.UpdateMeshDevSpace(&info); err != nil {
		log.Error(err)
		_ = meshManager.Rollback(&info)
		api.SendResponse(c, errno.ErrUpdateMeshSpaceFailed, nil)
		return
	}

	devspace.TraceHeader = info.Header
	result, err := service.Svc.ClusterUserSvc.Update(c, devspace)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrUpdateMeshSpaceFailed, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
