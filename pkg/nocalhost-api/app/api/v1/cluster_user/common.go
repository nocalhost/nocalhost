/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

// LoginUserHasModifyPermissionToSomeDevSpace
// those role can modify the devSpace:
// - is the cooperator of ns
// - is nocalhost admin
// - is owner
// - is devSpace's cluster owner
func LoginUserHasModifyPermissionToSomeDevSpace(c *gin.Context, devSpaceId uint64) (*model.ClusterUserModel, error) {

	loginUser, err := ginbase.LoginUser(c)
	if err != nil {
		return nil, errno.ErrPermissionDenied
	}

	return HasModifyPermissionToSomeDevSpace(loginUser, devSpaceId)
}

func HasModifyPermissionToSomeDevSpace(userId, devSpaceId uint64) (*model.ClusterUserModel, error) {
	devSpace, err := service.Svc.ClusterUserSvc.GetCache(devSpaceId)
	if err != nil {
		return nil, errno.ErrClusterUserNotFound
	}

	cluster, err := service.Svc.ClusterSvc.GetCache(devSpace.ClusterId)
	if err != nil {
		return nil, errno.ErrClusterNotFound
	}

	nss := ns_scope.AllCoopNs(devSpace.ClusterId, userId)

	for _, s := range nss {
		if devSpace.Namespace == s {
			return &devSpace, nil
		}
	}

	usr, err := service.Svc.UserSvc.GetUserByID(context.TODO(), userId)
	if err != nil {
		return nil, err
	}
	if (usr.IsAdmin != nil && *usr.IsAdmin == 1) || cluster.UserId == userId || devSpace.UserId == userId {
		return &devSpace, nil
	}

	return nil, errno.ErrPermissionDenied
}

// HasPrivilegeToSomeDevSpace
// Include
// - update resource limit
// - delete devspace
func HasPrivilegeToSomeDevSpace(c *gin.Context, devSpaceId uint64) (*model.ClusterUserModel, error) {
	devSpace, err := service.Svc.ClusterUserSvc.GetCache(devSpaceId)
	if err != nil {
		return nil, errno.ErrClusterUserNotFound
	}

	cluster, err := service.Svc.ClusterSvc.GetCache(devSpace.ClusterId)
	if err != nil {
		return nil, errno.ErrClusterNotFound
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
	shareSpaces, err := service.Svc.ClusterUserSvc.GetList(c, model.ClusterUserModel{BaseDevSpaceId: baseSpaceId})
	if err != nil {
		// can not find share space, do nothing
		return
	}
	for _, space := range shareSpaces {
		clusterData, err := service.Svc.ClusterSvc.GetCache(space.ClusterId)
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

func reCreateShareSpaces(c *gin.Context, user, baseSpaceId uint64) {
	shareSpaces, err := service.Svc.ClusterUserSvc.GetList(c, model.ClusterUserModel{BaseDevSpaceId: baseSpaceId})
	if err != nil {
		// can not find share space, do nothing
		return
	}

	for _, clusterUser := range shareSpaces {
		res := SpaceResourceLimit{}
		_ = json.Unmarshal([]byte(clusterUser.SpaceResourceLimit), &res)
		// create a new dev space
		meshDevInfo := &setupcluster.MeshDevInfo{
			Header:   clusterUser.TraceHeader,
			ReCreate: true,
		}
		req := ClusterUserCreateRequest{
			ClusterId:          &clusterUser.ClusterId,
			UserId:             &clusterUser.UserId,
			SpaceName:          clusterUser.SpaceName,
			Memory:             &clusterUser.Memory,
			Cpu:                &clusterUser.Cpu,
			ApplicationId:      &clusterUser.ApplicationId,
			NameSpace:          clusterUser.Namespace,
			ID:                 &clusterUser.ID,
			SpaceResourceLimit: &res,
			BaseDevSpaceId:     clusterUser.BaseDevSpaceId,
			MeshDevInfo:        meshDevInfo,
		}

		cluster, err := service.Svc.ClusterSvc.GetCache(clusterUser.ClusterId)
		if err != nil {
			log.Error(err)
			return
		}

		// delete devSpace space first, it will delete database record whatever success delete namespace or not
		devSpace := NewDevSpace(req, c, []byte(cluster.KubeConfig))

		list, e := DoList(&model.ClusterUserModel{ID: baseSpaceId}, user, ginbase.IsAdmin(c), false)
		if e != nil {
			log.Error(e)
			return
		}
		if len(list) != 1 {
			log.Errorf(errno.ErrClusterUserNotFound.Error())
			return
		}
		cu := list[0]

		if cu.IsClusterAdmin() {
			return
		}

		err = devSpace.Delete()
		if err != nil {
			log.Error(err)
			return
		}

		result, err := devSpace.Create()
		if err != nil {
			log.Error(err)
			return
		}

		if err := service.Svc.AuthorizeNsToUser(result.ClusterId, result.UserId, result.Namespace); err != nil {
			log.Error(err)
			return
		}

		if err := service.Svc.AuthorizeNsToDefaultSa(result.ClusterId, result.UserId, result.Namespace); err != nil {
			log.Error(err)
			return
		}

		for _, viewer := range cu.ViewerUser {
			_ = ns_scope.AsViewer(result.ClusterId, viewer.ID, result.Namespace)
		}

		for _, cooper := range cu.CooperUser {
			_ = ns_scope.AsCooperator(result.ClusterId, cooper.ID, result.Namespace)
		}
	}
}
