package cluster_user

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

func HasModifyPermissionToSomeDevSpace(c *gin.Context, devSpaceId uint64) (*model.ClusterUserModel, *errno.Errno) {
	devSpace, err := service.Svc.ClusterUser().GetCache(devSpaceId)
	if err != nil {
		return nil, errno.ErrClusterUserNotFound
	}

	cluster, err := service.Svc.ClusterSvc().GetCache(devSpace.ClusterId)
	if err != nil {
		return nil, errno.ErrClusterKubeErr
	}

	loginUser, err := ginbase.LoginUser(c)
	if err != nil {
		return nil, errno.ErrPermissionDenied
	}

	if ginbase.IsAdmin(c) || cluster.UserId == loginUser || devSpace.UserId == loginUser {
		return &devSpace, nil
	}
	return nil, errno.ErrPermissionDenied
}

func HasHighPermissionToSomeDevSpace(c *gin.Context, devSpaceId uint64) (*model.ClusterUserModel, *errno.Errno) {
	devSpace, err := service.Svc.ClusterUser().GetCache(devSpaceId)
	if err != nil {
		return nil, errno.ErrClusterUserNotFound
	}

	cluster, err := service.Svc.ClusterSvc().GetCache(devSpace.ClusterId)
	if err != nil {
		return nil, errno.ErrClusterKubeErr
	}

	loginUser, err := ginbase.LoginUser(c)
	if err != nil {
		return nil, errno.ErrPermissionDenied
	}

	if ginbase.IsAdmin(c) || cluster.UserId == loginUser {
		return &devSpace, nil
	}
	return nil, errno.ErrPermissionDenied
}

func IsShareUsersOk(cooperators, viewers []uint64, clusterUser *model.ClusterUserModel) bool {
	users := make([]uint64, len(cooperators)+len(viewers))
	copy(users, cooperators)
	copy(users[len(cooperators):], viewers)
	for i := range users {
		if users[i] == clusterUser.UserId {
			return false
		}
	}
	return true
}
