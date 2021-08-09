package cluster_user

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

func Share(c *gin.Context) {

	var params ClusterUserShareRequest
	err := c.ShouldBindBodyWith(&params, binding.JSON)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu, err := service.Svc.ClusterUser().GetCache(*params.ClusterUserId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	// the api to modify sharing RBAC is different
	// from whether is cluster admin
	if cu.ClusterAdmin != nil && *cu.ClusterAdmin != uint64(0) {
		for _, user := range params.Cooperators {
			if err := cluster_scope.AsCooperator(cu.ClusterId, cu.UserId, user); err != nil {
				log.ErrorE(err, "Error while add somebody as cluster cooperator")
			}
		}

		for _, user := range params.Viewers {
			if err := cluster_scope.AsViewer(cu.ClusterId, cu.UserId, user); err != nil {
				log.ErrorE(err, "Error while add somebody as cluster viewer")
			}
		}
	} else {
		for _, user := range params.Cooperators {
			if err := ns_scope.AsCooperator(cu.ClusterId, user, cu.Namespace); err != nil {
				log.ErrorE(err, "Error while add somebody as cooperator")
			}
		}

		for _, user := range params.Viewers {
			if err := ns_scope.AsViewer(cu.ClusterId, user, cu.Namespace); err != nil {
				log.ErrorE(err, "Error while add somebody as viewer")
			}
		}
	}

	api.SendResponse(c, nil, nil)
}

func UnShare(c *gin.Context) {
	var params ClusterUserUnShareRequest
	err := c.ShouldBindBodyWith(&params, binding.JSON)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu, err := service.Svc.ClusterUser().GetCache(*params.ClusterUserId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	// the api to modify sharing RBAC is different
	// from whether is cluster admin
	if cu.ClusterAdmin != nil && *cu.ClusterAdmin != uint64(0) {
		for _, user := range params.Users {
			if err := cluster_scope.RemoveFromCooperator(cu.ClusterId, cu.UserId, user); err != nil {
				log.ErrorE(err, "Error while remove somebody as cluster cooperator")
			}
			if err := cluster_scope.RemoveFromViewer(cu.ClusterId, cu.UserId, user); err != nil {
				log.ErrorE(err, "Error while remove somebody as cluster viewer")
			}
		}

	} else {
		for _, user := range params.Users {
			if err := ns_scope.RemoveFromViewer(cu.ClusterId, user, cu.Namespace); err != nil {
				log.ErrorE(err, "Error while remove somebody as viewer")
			}
			if err := ns_scope.RemoveFromCooperator(cu.ClusterId, user, cu.Namespace); err != nil {
				log.ErrorE(err, "Error while remove somebody as cooperator")
			}
		}
	}

	api.SendResponse(c, nil, nil)
}
