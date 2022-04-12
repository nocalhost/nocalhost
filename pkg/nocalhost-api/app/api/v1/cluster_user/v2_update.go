/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func Share(c *gin.Context) {

	var params ClusterUserShareRequest
	err := c.ShouldBindBodyWith(&params, binding.JSON)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu, errn := LoginUserHasModifyPermissionToSomeDevSpace(c, *params.ClusterUserId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	if !IsShareUsersOk(params.Cooperators, params.Viewers, cu) {
		api.SendResponse(c, errno.ErrShareUserSameAsOwner, nil)
		return
	}

	// the api to modify sharing RBAC is different
	// from whether is cluster admin
	if cu.IsClusterAdmin() {
		for _, user := range params.Cooperators {
			if err := cluster_scope.AsCooperator(cu.ClusterId, cu.UserId, user); err != nil {
				log.Errorf("Error while add somebody as cluster cooperator: %+v", err)
			}
		}

		for _, user := range params.Viewers {
			if err := cluster_scope.AsViewer(cu.ClusterId, cu.UserId, user); err != nil {
				log.Errorf("Error while add somebody as cluster viewer: %+v", err)
			}
		}
	} else {
		for _, user := range params.Cooperators {
			if err := ns_scope.AsCooperator(cu.ClusterId, user, cu.Namespace); err != nil {
				log.Errorf("Error while add somebody as cooperator: %+v", err)
			}
		}

		for _, user := range params.Viewers {
			if err := ns_scope.AsViewer(cu.ClusterId, user, cu.Namespace); err != nil {
				log.Errorf("Error while add somebody as viewer: %+v", err)
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

	cu, errn := LoginUserHasModifyPermissionToSomeDevSpace(c, *params.ClusterUserId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	// the api to modify sharing RBAC is different
	// from whether is cluster admin
	if cu.ClusterAdmin != nil && *cu.ClusterAdmin != uint64(0) {
		for _, user := range params.Users {
			if err := cluster_scope.RemoveFromCooperator(cu.ClusterId, cu.UserId, user); err != nil {
				log.Errorf("Error while remove somebody as cluster cooperator: %+v", err)
			}
			if err := cluster_scope.RemoveFromViewer(cu.ClusterId, cu.UserId, user); err != nil {
				log.Errorf("Error while remove somebody as cluster viewer: %+v", err)
			}
		}

	} else {
		for _, user := range params.Users {
			if err := ns_scope.RemoveFromViewer(cu.ClusterId, user, cu.Namespace); err != nil {
				log.Errorf("Error while remove somebody as viewer: %+v", err)
			}
			if err := ns_scope.RemoveFromCooperator(cu.ClusterId, user, cu.Namespace); err != nil {
				log.Errorf("Error while remove somebody as cooperator: %+v", err)
			}
		}
	}

	api.SendResponse(c, nil, nil)
}
