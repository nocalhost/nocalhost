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
	"time"
)

func Sleep(c *gin.Context) {
	// 1. obtain devspace
	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: id})
	if err != nil || space == nil {
		log.Errorf("Failed to resolve dev space, id = [%d]", id)
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	// 2. obtain cluster
	cluster, err := service.Svc.ClusterSvc().Get(c, space.ClusterId)
	if err != nil {
		log.Errorf("Failed to resolve cluster, id = [%d]", space.ClusterId)
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}
	// 3. obtain client-go
	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 4. force sleep
	err = sleep.Sleep(client, space.Namespace, true)
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 5. write to database
	now := time.Now().UTC()
	err = service.Svc.
		ClusterUser().
		Modify(c, id, map[string]interface{}{
			"SleepAt": &now,
			"IsAsleep": true,
		})
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrForceSleep, nil)
		return
	}
	// 6. response to HTTP
	api.SendResponse(c, errno.OK, nil)
}

func Wakeup(c *gin.Context) {
	// 1. obtain devspace
	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: id})
	if err != nil || space == nil {
		log.Errorf("Failed to resolve dev space, id = [%d]", id)
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	// 2. obtain cluster
	cluster, err := service.Svc.ClusterSvc().Get(c, space.ClusterId)
	if err != nil {
		log.Errorf("Failed to resolve cluster, id = [%d]", space.ClusterId)
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}
	// 3. obtain client-go
	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 4. force sleep
	err = sleep.Wakeup(client, space.Namespace, true)
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 5. write to database
	err = service.Svc.
		ClusterUser().
		Modify(c, id, map[string]interface{}{
			"SleepAt": nil,
			"IsAsleep": false,
		})
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrForceWakeup, nil)
		return
	}
	// 6. response to HTTP
	api.SendResponse(c, errno.OK, nil)
}

func UpdateSleepConfig(c *gin.Context) {
	// 1. binding
	var payload model.SleepConfig
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Errorf("Failed to resolve params, err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	// 2. obtain devspace
	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: id})
	if err != nil || space == nil {
		log.Errorf("Failed to resolve dev space, id = [%d]", id)
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	// 3. obtain cluster
	cluster, err := service.Svc.ClusterSvc().Get(c, space.ClusterId)
	if err != nil {
		log.Errorf("Failed to resolve cluster, id = [%d]", space.ClusterId)
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}
	// 4. obtain client-go
	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 5. update annotations
	if len(payload.Schedules) == 0 {
		err = sleep.DeleteSleepConfig(client, space.Namespace)
		if err != nil {
			api.SendResponse(c, err, nil)
			return
		}
	} else {
		marshal, _ := json.Marshal(payload)
		err = sleep.CreateSleepConfig(client, space.Namespace, string(marshal))
		if err != nil {
			api.SendResponse(c, err, nil)
			return
		}
	}
	// 6. write to database
	result, err := service.Svc.ClusterUser().Update(c, &model.ClusterUserModel{
		ID: id,
		SleepConfig: &payload,
	})
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrUpdateSleepConfig, nil)
		return
	}
	// 7. response to HTTP
	api.SendResponse(c, errno.OK, result)
}
