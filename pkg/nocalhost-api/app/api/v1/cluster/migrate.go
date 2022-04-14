/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/spf13/cast"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/api/v1/cluster_user"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/utils"
)

func Migrate(c *gin.Context) {
	var req ClusterUserMigrateRequest
	err := c.ShouldBindBodyWith(&req, binding.YAML)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	clusterId := cast.ToUint64(c.Param("id"))

	cluster, errn := HasPrivilegeToSomeCluster(c, clusterId)
	if errn != nil {
		api.SendResponse(c, errn, nil)
		return
	}

	userId, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	for _, m := range req.Migrate {

		defaultNum := uint64(0)

		devSpace := cluster_user.NewDevSpace(
			cluster_user.ClusterUserCreateRequest{
				ClusterId:    &cluster.ID,
				UserId:       &userId,
				SpaceName:    "auto_" + m.Namespace,
				NameSpace:    m.Namespace,
				Memory:       &defaultNum,
				Cpu:          &defaultNum,
				ClusterAdmin: &defaultNum,
				Protected:    true,
			}, c, []byte{},
		)

		_, err := devSpace.Create()
		if err != nil {
			log.ErrorE(err, "Fail to migrate namespace ")
		}

		for _, userEmail := range m.Users {
			if !utils.IsEmail(userEmail) {
				userEmail += _const.DefaultEmailSuffix
			}

			userPointer, err := service.Svc.UserSvc.CreateOrUpdateUserByEmail(
				c, userEmail, "",
				"", 0, false,
			)
			if err != nil {
				log.ErrorE(err, "Fail to migrate user ")
			}

			if err := ns_scope.AsCooperator(clusterId, userPointer.ID, m.Namespace); err != nil {
				log.ErrorE(err, "Fail to migrate user as cooper ")
				return
			}
		}
	}
}
