/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
}

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
	}
	result, err := service.Svc.ClusterUser().Update(c, &cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Update dev space
// @Description Update dev space
// @Tags Cluster
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
	devspace, err := service.Svc.ClusterUser().GetFirst(c, condition)
	if err != nil || devspace == nil {
		log.Errorf("Dev space has not found")
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	// Build goclient with administrator kubeconfig
	clusterData, err := service.Svc.ClusterSvc().Get(c, devspace.ClusterId)
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
	result, err := service.Svc.ClusterUser().Update(c, devspace)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Update mesh dev space info
// @Description Update mesh dev space info
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "devspace id"
// @Param MeshDevInfo body model.MeshDevInfo true "mesh dev space info"
// @Success 200 {object} model.ClusterUserModel
// @Router /v1/dev_space/{id}/update_resource_limit [put]
func UpdateMeshDevSpaceInfo(c *gin.Context) {
	var req model.MeshDevInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind resource limits params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	devSpaceId := cast.ToUint64(c.Param("id"))


	condition := model.ClusterUserModel{
		ID: devSpaceId,
	}
	devspace, err := service.Svc.ClusterUser().GetFirst(c, condition)
	if err != nil || devspace == nil {
		log.Errorf("Dev space has not found")
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	// Build goclient with administrator kubeconfig
	clusterData, err := service.Svc.ClusterSvc().Get(c, devspace.ClusterId)
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

	info := devspace.MeshDevInfo
	info.Header = req.Header
	info.APPS = req.APPS

	meshManager, err := setupcluster.NewMeshManager(goClient, info)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}

	if err := meshManager.InjectMeshDevSpace(); err != nil {
		api.SendResponse(c, nil, nil)
		return
	}

	devspace.MeshDevInfo = info
	result, err := service.Svc.ClusterUser().Update(c, devspace)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
