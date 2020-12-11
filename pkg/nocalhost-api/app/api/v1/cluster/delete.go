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

package cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
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

	cluster, err := service.Svc.ClusterSvc().Get(c, clusterId, userId.(uint64))
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	goClient, err := clientgo.NewGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		return
	}
	// delete nocalhost-reserved
	_, err = goClient.DeleteNS(global.NocalhostSystemNamespace)
	if err != nil {
		// ignore fail
		log.Warnf("delete cluster for id %s fail, err %s", clusterId, err.Error())
	}

	// get all dev space
	condition := model.ClusterUserModel{
		ClusterId: clusterId,
	}
	devSpace, err := service.Svc.ClusterUser().GetList(c, condition)
	var spaceIds []uint64
	if len(devSpace) > 0 {
		for _, space := range devSpace {
			_, _ = goClient.DeleteNS(space.Namespace)
			spaceIds = append(spaceIds, space.ID)
		}
	}

	// delete database cluster and cluster users
	err = service.Svc.ClusterSvc().Delete(c, clusterId)
	if err != nil {
		api.SendResponse(c, errno.ErrDeletedClsuterDBButClusterDone, nil)
		return
	}

	if len(spaceIds) > 0 {
		err = service.Svc.ClusterUser().BatchDelete(c, spaceIds)
		if err != nil {
			api.SendResponse(c, errno.ErrDeletedClsuterDevSpaceDBButClusterDone, nil)
			return
		}
	}

	api.SendResponse(c, errno.OK, nil)
}
