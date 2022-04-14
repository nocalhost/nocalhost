/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package user

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create Delete users
// @Summary Delete users
// @Description Delete users
// @Tags Users
// @Accept json
// @Produce json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "User ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/users/{id} [delete]
func Delete(c *gin.Context) {
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 {
		api.SendResponse(c, errno.ErrUpdateUserDenied, nil)
		return
	}
	userId := cast.ToUint64(c.Param("id"))
	// delete user's cluster dev space first
	condition := model.ClusterUserJoinCluster{
		UserId: userId,
	}
	var clusterUserIds []uint64
	clusterUserList, err := service.Svc.ClusterUserSvc.GetJoinCluster(c, condition)
	if len(clusterUserList) > 0 {
		for _, clusterUser := range clusterUserList {
			goClient, err := clientgo.NewAdminGoClient([]byte(clusterUser.AdminClusterKubeConfig))
			if err != nil {
				log.Warnf("try to delete userid %d while create go-client fail", clusterUser.UserId)
				continue
			}
			_, err = goClient.DeleteNS(clusterUser.Namespace)
			if err != nil {
				log.Warnf(
					"try to delete userid %d cluster namesapce %d fail", clusterUser.UserId, clusterUser.Namespace,
				)
			}
			log.Infof("deleted user cluster dev space %s", clusterUser.Namespace)
			clusterUserIds = append(clusterUserIds, clusterUser.ID)
		}
	}

	user, err := service.Svc.UserSvc.GetCache(userId)
	if err != nil {
		log.Warnf("user delete error: %v", err)
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	clusterList, err := service.Svc.ClusterSvc.GetList(c)
	if err != nil {
		log.Warnf("user delete error: %v", err)
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	for _, clusterItem := range clusterList {
		cl, err := service.Svc.ClusterSvc.GetCache(clusterItem.ID)
		if err != nil {
			log.Error(err)
			continue
		}

		// new client go
		clientGo, err := clientgo.NewAdminGoClient([]byte(cl.KubeConfig))
		if err != nil {
			log.Error(err)
			continue
		}

		if err := clientGo.DeleteServiceAccount(user.SaName, _const.NocalhostDefaultSaNs); err != nil {
			log.Error(err)
			continue
		}
	}

	// delete cluster user database record
	err = service.Svc.ClusterUserSvc.BatchDelete(c, clusterUserIds)
	if err != nil {
		log.Warnf("try to delete dev spaceId %s fail", clusterUserIds)
	}

	err = service.Svc.UserSvc.Delete(c, userId)
	if err != nil {
		log.Warnf("user delete error: %v", err)
		api.SendResponse(c, errno.ErrDeleteUser, nil)
		return
	}

	// if delete normal user, needs to delete cluster which added by this user
	if user.IsAdmin != nil && *user.IsAdmin != 1 {
		err = service.Svc.ClusterSvc.DeleteByCreator(c, userId)
		if err != nil {
			log.Warnf("delete cluster which created by this user error: %v", err)
			api.SendResponse(c, errno.ErrDeleteUser, nil)
			return
		}
	}

	api.SendResponse(c, errno.OK, nil)
}
