package cluster_user

import (
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

// Sleep
// @Summary Sleep
// @Description Forced dev space into sleep mode
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "dev space id"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v2/dev_space/{id}/sleep [post]
func Sleep(c *gin.Context) {
	// 1. obtain dev space
	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetCache(id)
	if err != nil {
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
	// 3. init client-go
	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 4. force sleep
	err = sleep.Asleep(client, &space, true)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrForceSleep, nil)
		return
	}
	// 5. response to HTTP
	api.SendResponse(c, errno.OK, nil)
}

// Wakeup
// @Summary Wakeup
// @Description Forced awakens the dev space
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "dev space id"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v2/dev_space/{id}/wakeup [post]
func Wakeup(c *gin.Context) {
	// 1. obtain dev space
	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetCache(id)
	if err != nil {
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
	// 3. init client-go
	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 4. force sleep
	err = sleep.Wakeup(client, &space, true)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrForceWakeup, nil)
		return
	}
	// 5. response to HTTP
	api.SendResponse(c, errno.OK, nil)
}

// ApplySleepConfig
// @Summary UpdateSleepConfig
// @Description update sleep config in dev space
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "dev space id"
// @Param sleep_config body model.SleepConfig true "Sleep Config"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v2/dev_space/{id}/sleep_config [put]
func ApplySleepConfig(c *gin.Context) {
	// 1. binding
	var payload model.SleepConfig
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Errorf("Failed to resolve params, err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	// 2. obtain devspace
	id := cast.ToUint64(c.Param("id"))
	space, err := service.Svc.ClusterUser().GetCache(id)
	if err != nil {
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
	// 4. init client-go
	client, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	// 5. update annotations
	err = sleep.Update(client, space.ID, space.Namespace, payload)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrUpdateSleepConfig, nil)
		return
	}
	// 6. response to HTTP
	api.SendResponse(c, errno.OK, nil)
}
