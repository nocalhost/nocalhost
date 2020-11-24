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
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Delete 彻底删除开发环境
// @Summary 彻底删除开发环境
// @Description 彻底删除开发环境，包含删除 K8S 命名空间
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "开发环境 ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/dev_space/{id} [delete]
func Delete(c *gin.Context) {
	userId, _ := c.Get("userId")
	devSpaceId := cast.ToUint64(c.Param("id"))
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClsuterUserNotFound, nil)
		return
	}
	clusterData, err := service.Svc.ClusterSvc().Get(c, clusterUser.ClusterId, userId.(uint64))
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	// client go and delete specify namespace
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewGoClient(KubeConfig)
	if err != nil {
		log.Errorf("client go got err %v", err)
		api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		return
	}
	isDelete, _ := goClient.DeleteNS(clusterUser.Namespace)
	if !isDelete {
		api.SendResponse(c, errno.ErrDeleteClusterNameSpace, nil)
		return
	}

	// delete database cluster-user dev space
	dErr := service.Svc.ClusterUser().Delete(c, clusterUser.ID)
	if dErr != nil {
		api.SendResponse(c, errno.ErrDeletedClsuterButDatabaseFail, nil)
		return
	}
	api.SendResponse(c, errno.OK, nil)
	return
}
