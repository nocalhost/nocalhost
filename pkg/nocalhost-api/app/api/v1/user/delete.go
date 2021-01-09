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

package user

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

// Create Delete users
// @Summary Delete users
// @Description Delete users
// @Tags Users
// @Accept json
// @Produce json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "User ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/users/{id} [delete]
func Delete(c *gin.Context) {
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		api.SendResponse(c, errno.ErrUpdateUserDenied, nil)
		return
	}
	userId := cast.ToUint64(c.Param("id"))
	// delete user's cluster dev space first
	condition := model.ClusterUserJoinCluster{
		UserId: userId,
	}
	var clusterUserIds []uint64
	clusterUserList, err := service.Svc.ClusterUser().GetJoinCluster(c, condition)
	if len(clusterUserList) > 0 {
		for _, clusterUser := range clusterUserList {
			goClient, err := clientgo.NewAdminGoClient([]byte(clusterUser.AdminClusterKubeConfig))
			if err != nil {
				log.Warnf("try to delete userid %d while create go-client fail", clusterUser.UserId)
				continue
			}
			_, err = goClient.DeleteNS(clusterUser.Namespace)
			if err != nil {
				log.Warnf("try to delete userid %d cluster namesapce %d fail", clusterUser.UserId, clusterUser.Namespace)
			}
			log.Infof("deleted user cluster dev space %s", clusterUser.Namespace)
			clusterUserIds = append(clusterUserIds, clusterUser.ID)
		}
	}

	// delete cluster user database record
	err = service.Svc.ClusterUser().BatchDelete(c, clusterUserIds)
	if err != nil {
		log.Warnf("try to delete dev spaceId %s fail", clusterUserIds)
	}

	err = service.Svc.UserSvc().Delete(c, userId)
	if err != nil {
		log.Warnf("user delete error: %v", err)
		api.SendResponse(c, errno.ErrDeleteUser, nil)
		return
	}

	api.SendResponse(c, errno.OK, nil)
}
