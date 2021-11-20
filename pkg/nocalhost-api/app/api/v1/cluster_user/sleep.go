package cluster_user

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/sleep"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func Sleep(c *gin.Context) {
	// TODO: sleep
}

func Wakeup(c *gin.Context) {
	// TODO: sleep
}

func UpdateSleepConfig(c *gin.Context) {
	var payload SleepConfig
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Warnf("Failed to resolve params, err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: id})
	if err != nil || space == nil {
		log.Errorf("Failed to resolve dev space, id = [%d]", id)
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	cluster, err := service.Svc.ClusterSvc().Get(c, space.ClusterId)
	if err != nil {
		log.Errorf("Failed to resolve cluster, id = [%d]", space.ClusterId)
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}

	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	marshal, _ := json.Marshal(payload)
	if len(payload.Rules) == 0 {
		err = sleep.DeleteSleepConfig(client, space.Namespace)
		if err != nil {
			api.SendResponse(c, err, nil)
			return
		}
	} else {
		err = sleep.CreateSleepConfig(client, space.Namespace, string(marshal))
		if err != nil {
			api.SendResponse(c, err, nil)
			return
		}
	}

	result, err := service.Svc.ClusterUser().Update(c, &model.ClusterUserModel{
		ID: space.ID,
		SleepConfig: string(marshal),
	})
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrUpdateSleepConfig, nil)
		return
	}

	api.SendResponse(c, nil, result)
}
