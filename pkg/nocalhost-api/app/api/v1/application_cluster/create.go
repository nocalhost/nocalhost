package application_cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create 关联集群
// @Summary 关联集群
// @Description 应用关联集群
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body application_cluster.ApplicationClusterRequest true "The application info"
// @Param id path uint64 true "应用 ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/application/{id}/bind_cluster [post]
func Create(c *gin.Context) {
	var req ApplicationClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind ApplicationCluster params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	// check application
	if _, err := service.Svc.ApplicationSvc().Get(c, applicationId, userId.(uint64)); err != nil {
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}
	// check cluster
	if _, err := service.Svc.ClusterSvc().Get(c, *req.ClusterId, userId.(uint64)); err != nil {
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}
	err := service.Svc.ApplicationClusterSvc().Create(c, applicationId, *req.ClusterId, userId.(uint64))
	if err != nil {
		log.Warnf("create ApplicationCluster err: %v", err)
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}
