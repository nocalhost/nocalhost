/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"sort"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

func GetV2(c *gin.Context) {
	var params ClusterUserGetRequest

	if err := c.ShouldBindQuery(&params); err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu := model.ClusterUserModel{}
	userId, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	isAdmin := ginbase.IsAdmin(c)
	if params.ClusterUserId == nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu.ID = *params.ClusterUserId
	result, err := DoList(&cu, userId, isAdmin, false)
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	// set base space name
	for i := range result {
		if result[i].BaseDevSpaceId > 0 {
			cu, err := service.Svc.ClusterUserSvc.GetFirst(c, model.ClusterUserModel{ID: result[i].BaseDevSpaceId})
			if err != nil {
				api.SendResponse(c, errno.ErrMeshClusterUserNotFound, nil)
				return
			}
			if result[i].ClusterUserExt == nil {
				result[i].ClusterUserExt = &model.ClusterUserExt{
					BaseDevSpaceName:      cu.SpaceName,
					BaseDevSpaceNameSpace: cu.Namespace,
				}
			} else {
				result[i].BaseDevSpaceName = cu.SpaceName
				result[i].BaseDevSpaceNameSpace = cu.Namespace
			}
		}
	}
	api.SendResponse(c, nil, result)
}

func ListV2(c *gin.Context) {
	var params ClusterUserListV2Query

	if err := c.ShouldBindQuery(&params); err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu := model.ClusterUserModel{}
	userId, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
		return
	}

	isAdmin := ginbase.IsAdmin(c)

	if params.OwnerUserId != nil {
		cu.UserId = *params.OwnerUserId
	}
	if params.ClusterId != nil {
		cu.ClusterId = *params.ClusterId
	}
	if params.SpaceName != "" {
		cu.SpaceName = params.SpaceName
	}

	if result, err := DoList(&cu, userId, isAdmin, params.IsCanBeUsedAsBaseSpace); err != nil {
		if err == errno.ErrClusterNotFound {
			log.Error(err)
			api.SendResponse(c, nil, []model.ClusterUserV2{})
			return
		}
		api.SendResponse(c, err, nil)
	} else {
		api.SendResponse(c, nil, result)
	}
}

func DoList(params *model.ClusterUserModel, userId uint64, isAdmin, isCanBeUsedAsBaseSpace bool) (
	[]*model.ClusterUserV2, error) {

	clusterUsers, err := service.Svc.ClusterUserSvc.ListV2(*params)
	if err != nil {
		log.Error(err)
		return nil, errno.ErrClusterNotFound
	}

	if errn := PipeLine(clusterUsers, userId, isAdmin); errn != nil {
		log.Error(err)
		return nil, errn
	}
	// user that not admin can not see other user's data
	if !isAdmin {
		// todo supports search for SpaceType(only need to deal with filter)
		clusterUsers = filter(clusterUsers, relatedToSomebody(userId))
	}

	// filter can be used for base space
	if isCanBeUsedAsBaseSpace && len(clusterUsers) > 0 {
		clusterUsers = filter(clusterUsers, isCanBeUsedAsBaseSpaceFun())
	}
	return clusterUsers, nil
}

func relatedToSomebody(userId uint64) func(*model.ClusterUserV2) bool {
	return func(cu *model.ClusterUserV2) bool {
		if cu.UserId == userId {
			return true
		}

		for _, cooperUser := range cu.CooperUser {
			if cooperUser.ID == userId {
				return true
			}
		}

		for _, viewerUser := range cu.ViewerUser {
			if viewerUser.ID == userId {
				return true
			}
		}

		clusterCache, err := service.Svc.ClusterSvc.GetCache(cu.ClusterId)
		if err != nil {
			return false
		}
		if clusterCache.UserId == userId {
			return true
		}

		return false
	}
}

func isCanBeUsedAsBaseSpaceFun() func(*model.ClusterUserV2) bool {
	return func(cu *model.ClusterUserV2) bool {
		return cu.SpaceType != model.MeshSpace && cu.IsBaseSpace
	}
}

func filter(clusterUsers []*model.ClusterUserV2, condition func(*model.ClusterUserV2) bool) []*model.ClusterUserV2 {
	result := make([]*model.ClusterUserV2, 0)
	for _, cu := range clusterUsers {
		if condition(cu) {
			result = append(result, cu)
		}
	}

	return result
}

func PipeLine(clusterUsers []*model.ClusterUserV2, userId uint64, isAdmin bool) error {
	// First group DevSpace by cluster and dispatch the RBAC via serviceAccount
	// associate by the current user
	// Then Filling the ext custom field for current user
	// Last sort the list
	// (For different user, may has different ext field and item's sort priority1)
	if err := fillExtByUser(groupByCLuster(clusterUsers), userId, isAdmin); err != nil {
		return err
	}
	doSort(clusterUsers)
	return nil
}

func doSort(clusterUsers []*model.ClusterUserV2) {
	sort.Slice(
		clusterUsers, func(i, j int) bool {
			cu1 := clusterUsers[i]
			cu2 := clusterUsers[j]
			//clusterAdmin show at the top
			if *cu1.ClusterAdmin > *cu2.ClusterAdmin {
				return true
			}
			if *cu1.ClusterAdmin < *cu2.ClusterAdmin {
				return false
			}
			if cu1.ClusterUserExt.SpaceOwnType.Priority > cu2.ClusterUserExt.SpaceOwnType.Priority {
				return true
			}
			if cu1.ClusterUserExt.SpaceOwnType.Priority < cu2.ClusterUserExt.SpaceOwnType.Priority {
				return false
			}
			if cu1.ClusterId < cu2.ClusterId {
				return true
			}
			if cu1.ClusterId > cu2.ClusterId {
				return false
			}
			if cu1.UserId < cu2.UserId {
				return true
			}
			if cu1.UserId > cu2.UserId {
				return false
			}
			return cu1.SpaceName < cu2.SpaceName
		},
	)
}

func fillExtByUser(src map[uint64][]*model.ClusterUserV2, currentUser uint64, isAdmin bool) error {
	list, err := service.Svc.ClusterSvc.GetList(context.TODO())
	if err != nil {
		log.Errorf("Error while list cluster: %+v", err)
		return errno.ErrClusterNotFound
	}

	// specify the SpaceOwnType according to the sa type in those ns
	// and transfer all sa to user info
	for _, cluster := range list {
		ownNss := ns_scope.AllOwnNs(cluster.ID, currentUser)
		coopNss := ns_scope.AllCoopNs(cluster.ID, currentUser)
		viewNss := ns_scope.AllViewNs(cluster.ID, currentUser)

		if cus, ok := src[cluster.ID]; ok {
			for _, cu := range cus {

				cu.ClusterUserExt = &model.ClusterUserExt{}
				fillResourceListSet(cu)
				fillOwner(cu)

				cu.ClusterName = cluster.ClusterName
				cu.Modifiable =
					isAdmin ||
						// current user is the owner of dev space
						cu.UserId == currentUser ||
						// current user is the creator of dev space's cluster
						cluster.UserId == currentUser

				cu.Deletable = isAdmin ||
					// current user is the creator of dev space's cluster
					cluster.UserId == currentUser

				if cu.IsClusterAdmin() {
					if cu.BaseDevSpaceId > 0 {
						cu.SpaceType = model.MeshSpace
					} else {
						cu.SpaceType = model.IsolateSpace
					}

					if cluster_scope.IsValidOwner(cluster.ID, currentUser) {
						cu.SpaceOwnType = model.DevSpaceOwnTypeOwner
					} else if cluster_scope.IsCooperAs(cluster.ID, cu.UserId, currentUser) {
						cu.SpaceOwnType = model.DevSpaceOwnTypeCooperator
					} else if cluster_scope.IsViewerAs(cluster.ID, cu.UserId, currentUser) {
						cu.SpaceOwnType = model.DevSpaceOwnTypeViewer
					} else {
						cu.SpaceOwnType = model.None
					}

					fillClusterCooperator(cu, cluster.ID)
					fillClusterViewer(cu, cluster.ID)
				} else {
					if cu.BaseDevSpaceId > 0 {
						cu.SpaceType = model.MeshSpace
					} else {
						cu.SpaceType = model.IsolateSpace
					}

					// fill SpaceOwnType
					if contains(ownNss, cu.Namespace) {
						cu.SpaceOwnType = model.DevSpaceOwnTypeOwner
					} else if contains(coopNss, cu.Namespace) {
						cu.SpaceOwnType = model.DevSpaceOwnTypeCooperator
					} else if contains(viewNss, cu.Namespace) {
						cu.SpaceOwnType = model.DevSpaceOwnTypeViewer
					} else {
						cu.SpaceOwnType = model.None
					}

					fillCooperator(cu, cluster.ID)
					fillViewer(cu, cluster.ID)
				}
			}
		}
	}

	return nil
}

func contains(arr []string, tar string) bool {
	for _, s := range arr {
		if s == tar {
			return true
		}
	}
	return false
}

func groupByCLuster(clusterUsers []*model.ClusterUserV2) map[uint64][]*model.ClusterUserV2 {
	var result = make(map[uint64][]*model.ClusterUserV2, 0)
	for _, cu := range clusterUsers {
		if _, ok := result[cu.ClusterId]; !ok {
			result[cu.ClusterId] = make([]*model.ClusterUserV2, 0)
		}

		result[cu.ClusterId] = append(result[cu.ClusterId], cu)
	}
	return result
}

func fillViewer(cu *model.ClusterUserV2, clusterId uint64) {
	viewerSa := ns_scope.ViewerSas(clusterId, cu.Namespace)
	cu.ViewerUser = make([]*model.UserSimple, 0)
	for _, sa := range viewerSa {
		usr, err := service.Svc.UserSvc.GetCacheBySa(sa)
		if err != nil {
			log.Error("Error while Get user cache by sa %s", sa)
			continue
		}
		cu.ViewerUser = append(cu.ViewerUser, usr.ToUserSimple())
	}
}

func fillClusterViewer(cu *model.ClusterUserV2, clusterId uint64) {
	cooperSa := cluster_scope.ViewSas(clusterId, cu.UserId)
	cu.ViewerUser = make([]*model.UserSimple, 0)
	for _, sa := range cooperSa {
		usr, err := service.Svc.UserSvc.GetCacheBySa(sa)
		if err != nil {
			log.Error("Error while Get user cache by sa %s", sa)
			continue
		}
		cu.ViewerUser = append(cu.ViewerUser, usr.ToUserSimple())
	}
}

func fillCooperator(cu *model.ClusterUserV2, clusterId uint64) {
	cooperSa := ns_scope.CoopSas(clusterId, cu.Namespace)
	cu.CooperUser = make([]*model.UserSimple, 0)
	for _, sa := range cooperSa {
		usr, err := service.Svc.UserSvc.GetCacheBySa(sa)
		if err != nil {
			log.Error("Error while Get user cache by sa %s", sa)
			continue
		}
		cu.CooperUser = append(cu.CooperUser, usr.ToUserSimple())
	}
}

func fillClusterCooperator(cu *model.ClusterUserV2, clusterId uint64) {
	cooperSa := cluster_scope.CoopSas(clusterId, cu.UserId)
	cu.CooperUser = make([]*model.UserSimple, 0)
	for _, sa := range cooperSa {
		usr, err := service.Svc.UserSvc.GetCacheBySa(sa)
		if err != nil {
			log.Error("Error while Get user cache by sa %s", sa)
			continue
		}
		cu.CooperUser = append(cu.CooperUser, usr.ToUserSimple())
	}
}

func fillOwner(cu *model.ClusterUserV2) {
	usr, err := service.Svc.UserSvc.GetCache(cu.UserId)
	if err != nil {
		log.Error("Error while Get user cache by id %s", cu.UserId)
		return
	}
	cu.Owner = usr.ToUserSimple()
}

func fillResourceListSet(
	cu *model.ClusterUserV2,
) {
	res := &SpaceResourceLimit{}
	err := json.Unmarshal([]byte(cu.SpaceResourceLimit), res)
	if err != nil || res == nil {
		cu.ResourceLimitSet = false
	}

	cu.ResourceLimitSet = res.ResourceLimitIsSet()
}
