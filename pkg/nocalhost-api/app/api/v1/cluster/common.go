package cluster

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// HasPrivilegeToSomeCluster
// include
// - * devspace
// - * cluster
func HasPrivilegeToSomeCluster(c *gin.Context, clusterId uint64) (*model.ClusterModel, error) {
	cluster, err := service.Svc.ClusterSvc().GetCache(clusterId)
	if err != nil {
		return nil, errno.ErrClusterNotFound
	}

	loginUser, err := ginbase.LoginUser(c)
	if err != nil {
		return nil, errno.ErrPermissionDenied
	}

	if ginbase.IsAdmin(c) || cluster.UserId == loginUser {
		return &cluster, nil
	}
	return nil, errno.ErrPermissionDenied
}
