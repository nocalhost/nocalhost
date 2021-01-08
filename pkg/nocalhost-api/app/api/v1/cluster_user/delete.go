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

package cluster_user

import (
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
	// userId, _ := c.Get("userId")
	devSpaceId := cast.ToUint64(c.Param("id"))
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClsuterUserNotFound, nil)
		return
	}

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
	err = devSpace.Delete()
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	api.SendResponse(c, errno.OK, nil)
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
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClsuterUserNotFound, nil)
		return
	}

	clusterData, err := service.Svc.ClusterSvc().Get(c, clusterUser.ClusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	// create a new dev space
	req := ClusterUserCreateRequest{
		ClusterId:     &clusterUser.ClusterId,
		UserId:        &clusterUser.UserId,
		SpaceName:     clusterUser.SpaceName,
		Memory:        &clusterUser.Memory,
		Cpu:           &clusterUser.Cpu,
		ApplicationId: &clusterUser.ApplicationId,
		NameSpace:     clusterUser.Namespace,
		ID:            &clusterUser.ID,
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
		api.SendResponse(c, errno.ErrClsuterUserNotFound, nil)
		return
	}
	ReCreate(c)
}
