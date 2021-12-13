package cluster_user

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

// HasModifyPermissionToSomeDevSpace
// those role can modify the devSpace:
// - is the cooperator of ns
// - is nocalhost admin
// - is owner
// - is devSpace's cluster owner
func HasModifyPermissionToSomeDevSpace(c *gin.Context, devSpaceId uint64) (*model.ClusterUserModel, error) {
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

	nss := ns_scope.AllCoopNs(devSpace.ClusterId, loginUser)

	for _, s := range nss {
		if devSpace.Namespace == s {
			return &devSpace, nil
		}
	}

	if ginbase.IsAdmin(c) || cluster.UserId == loginUser || devSpace.UserId == loginUser {
		return &devSpace, nil
	}
	return nil, errno.ErrPermissionDenied
}

// HasHighPermissionToSomeDevSpace
// High Permission include
// - update resource limit
// - delete devspace
func HasHighPermissionToSomeDevSpace(c *gin.Context, devSpaceId uint64) (*model.ClusterUserModel, error) {
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

func deleteShareSpaces(c *gin.Context, baseSpaceId uint64) {
	shareSpaces, err := service.Svc.ClusterUser().GetList(c, model.ClusterUserModel{BaseDevSpaceId: baseSpaceId})
	if err != nil {
		// can not find share space, do nothing
		return
	}
	for _, space := range shareSpaces {
		clusterData, err := service.Svc.ClusterSvc().GetCache(space.ClusterId)
		if err != nil {
			continue
		}
		meshDevInfo := &setupcluster.MeshDevInfo{
			Header: space.TraceHeader,
		}
		req := ClusterUserCreateRequest{
			ID:             &space.ID,
			NameSpace:      space.Namespace,
			BaseDevSpaceId: space.BaseDevSpaceId,
			MeshDevInfo:    meshDevInfo,
		}
		_ = NewDevSpace(req, c, []byte(clusterData.KubeConfig)).Delete()
	}
}
