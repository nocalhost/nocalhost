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

package applications

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

// Create 删除应用
// @Summary 删除应用
// @Description 用户删除应用，同时也会删除应用内已配置的开发空间
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "应用 ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id} [delete]
func Delete(c *gin.Context) {
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	// get namespace
	condition := model.ClusterUserJoinCluster{
		ApplicationId: applicationId,
	}
	clusterUserList, err := service.Svc.ClusterUser().GetJoinCluster(c, condition)
	if err != nil || len(clusterUserList) == 0 {
		// has not dev space
		api.SendResponse(c, errno.OK, nil)
		return
	}

	// delete dev Namespace
	var spaceIds []uint64
	if len(clusterUserList) > 0 {
		for _, devSpace := range clusterUserList {
			goClient, err := clientgo.NewGoClient([]byte(devSpace.AdminClusterKubeConfig))
			if err != nil {
				// ignore and continus
				continue
			}
			_, err = goClient.DeleteNS(devSpace.Namespace)
			if err != nil {
				log.Warnf("delete application's cluster namespace %s fail, ignore", devSpace.Namespace)
			}
			log.Infof("deleted application's cluster namespace %s", devSpace.Namespace)
			spaceIds = append(spaceIds, devSpace.ID)
		}
	}

	// delete dev space database record
	if len(spaceIds) > 0 {
		err = service.Svc.ClusterUser().BatchDelete(c, spaceIds)
		if err != nil {
			api.SendResponse(c, errno.ErrDeletedClusterRecord, nil)
			return
		}
	}

	// delete application database record
	err = service.Svc.ApplicationSvc().Delete(c, userId.(uint64), applicationId)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationDelete, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
