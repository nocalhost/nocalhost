/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/api/v1/service_account"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

// @Summary Plug-in Get personal application development environment (kubeconfig) (obsolete)
// @Description Get personal application development environment (kubeconfig) (obsolete)
// @Tags Plug-in
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Application ID"
// @Success 200 {object} model.ClusterUserModel "Application
// development environment parameters, including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/application/{id}/dev_space [get]
func GetFirst(c *gin.Context) {
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId.(uint64),
	}
	result, err := service.Svc.ClusterUserSvc.GetFirst(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get a list of application development environments
// @Description Get application dev space list
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Application ID"
// @Success 200 {object} model.ClusterUserModel "Application development environment parameters,
//including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/application/{id}/dev_space_list [get]
func GetList(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
	}
	result, err := service.Svc.ClusterUserSvc.GetList(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// ListAll dev spaces
// @Summary ListAll dev spaces
// @Description ListAll dev spaces
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.ClusterUserModel
// @Router /v1/dev_space/{id} [get]
func ListAll(c *gin.Context) {
	var params ClusterUserListQuery

	err := c.ShouldBindQuery(&params)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	cu := model.ClusterUserModel{}

	if ginbase.IsAdmin(c) {
		if params.UserId != nil {
			cu.UserId = *params.UserId
		}
	} else {
		user, _ := ginbase.LoginUser(c)
		cu.UserId = user
	}

	result, err := service.Svc.ClusterUserSvc.GetList(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// list user's dev space distinct by user id
func ListByUserId(c *gin.Context) {
	userId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ClusterUserSvc.ListByUser(c, userId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	list, err := service.Svc.ClusterSvc.GetList(context.TODO())
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	set := map[uint64]*model.ClusterList{}
	for _, c := range list {
		set[c.ID] = c
	}

	for _, r := range result {
		c, ok := set[r.ClusterId]

		if ok {
			r.StorageClass = c.StorageClass
		}

		r.DevStartAppendCommand = []string{
			global.NocalhostDefaultPriorityclassKey, global.NocalhostDefaultPriorityclassName,
		}
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get the details of a development environment of the application
// @Description Get dev space detail from application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Application ID"
// @Param space_id path string true "DevSpace ID"
// @Success 200 {object} model.ClusterUserModel "Application development environment parameters,
//including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/application/{id}/dev_space/{space_id}/detail [get]
func GetDevSpaceDetail(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	spaceId := cast.ToUint64(c.Param("space_id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		ID:            spaceId,
	}
	result, err := service.Svc.ClusterUserSvc.GetFirst(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get a list of application development environments
// @Description Get application dev space list
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "User ID"
// @Success 200 {object} model.ClusterUserJoinClusterAndAppAndUser "Application development environment parameters,
//including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/users/{id}/dev_space_list [get]
func GetJoinClusterAndAppAndUser(c *gin.Context) {
	condition := model.ClusterUserJoinClusterAndAppAndUser{}
	userId := cast.ToUint64(c.Param("id"))
	userIdContext, _ := c.Get("userId")
	isAdmin, _ := c.Get("isAdmin")
	if isAdmin.(uint64) != 1 { // The developer queries devspace
		condition.UserId = cast.ToUint64(userIdContext)
	} else if userId != cast.ToUint64(userIdContext) { // The administrator queries the designated devspace
		condition.UserId = userId
	}

	result, err := service.Svc.ClusterUserSvc.GetJoinClusterAndAppAndUser(c, condition)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get the details of a development environment of the application
// @Description Get dev space detail from application
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "DevSpace ID"
// @Success 200 {object} model.ClusterUserJoinClusterAndAppAndUser "Application development environment parameters,
//including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/dev_space/{id}/detail [get]
func GetJoinClusterAndAppAndUserDetail(c *gin.Context) {
	condition := model.ClusterUserJoinClusterAndAppAndUser{
		ID: cast.ToUint64(c.Param("id")),
	}

	var params ClusterUserListQuery
	err := c.ShouldBindQuery(&params)
	if err != nil {
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	user, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionDenied, nil)
	}

	result, err := service.Svc.ClusterUserSvc.GetJoinClusterAndAppAndUserDetail(c, condition)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, errn := HasPrivilegeToSomeDevSpace(c, result.ID)
	var noPermissionToViewOtherKubeconfig = false
	if errn != nil {
		if errn != errno.ErrPermissionDenied {
			api.SendResponse(c, errn, nil)
			return
		}

		noPermissionToViewOtherKubeconfig = true
	}

	userChan := make(chan *model.UserBaseModel, 1)
	clusterChan := make(chan model.ClusterPack, 1)
	configMapChan := make(chan string, 1)

	defer func() {
		close(userChan)
		close(clusterChan)
		close(configMapChan)
	}()

	go func() {
		var queryUser uint64
		if params.UserId == nil {
			queryUser = result.UserId
		} else {
			queryUser = *params.UserId
		}

		if noPermissionToViewOtherKubeconfig {
			queryUser = user
		}

		userRecord, err := service.Svc.UserSvc.GetUserByID(c, queryUser)
		if err != nil {
			return
		}

		userChan <- userRecord
	}()

	go func() {
		clusterRecord, err := service.Svc.ClusterSvc.Get(c, result.ClusterId)
		if err != nil {
			return
		}

		clusterChan <- &clusterRecord
	}()

	go func() {
		userModel := <-userChan
		pack := <-clusterChan

		service_account.GenKubeconfig(
			userModel.SaName, pack, result.Namespace,
			func(nss []service_account.NS, privilegeType service_account.PrivilegeType, kubeConfig string) {
				configMapChan <- kubeConfig
			},
		)
	}()

	select {
	case <-ctx.Done():
		api.SendResponse(c, errno.InternalServerTimeoutError, nil)
	case kubeConfig, ok := <-configMapChan:
		if ok {
			result.KubeConfig = kubeConfig
			api.SendResponse(c, nil, result)
		} else {
			api.SendResponse(c, errno.InternalServerError, nil)
		}
	}
}

// GetAppsInfo
// @Summary Get mesh apps info
// @Description Get mesh apps info
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "devspace id"
// @Success 200 {object} setupcluster.MeshDevInfo
// @Router /v1/dev_space/{id}/mesh_apps_info [get]
func GetAppsInfo(c *gin.Context) {
	devSpaceId := cast.ToUint64(c.Param("id"))

	condition := model.ClusterUserModel{
		ID: devSpaceId,
	}
	devspace, err := service.Svc.ClusterUserSvc.GetFirst(c, condition)
	if err != nil || devspace == nil {
		log.Errorf("Dev space has not found")
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}
	isBasespace := devspace.BaseDevSpaceId == 0

	// check base dev space
	basespace := &model.ClusterUserModel{}
	if !isBasespace {
		baseCondition := model.ClusterUserModel{
			ID: devspace.BaseDevSpaceId,
		}
		basespace, err = service.Svc.ClusterUserSvc.GetFirst(c, baseCondition)
		if err != nil || basespace == nil {
			log.Errorf("Base space has not found")
			api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
			return
		}
	}

	// Build goclient with administrator kubeconfig
	clusterData, err := service.Svc.ClusterSvc.Get(c, devspace.ClusterId)
	if err != nil {
		log.Errorf("Getting cluster information failed, cluster id = [ %v ] ", devspace.ClusterId)
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}

	meshManager, err := setupcluster.GetSharedMeshManagerFactory().Manager(clusterData.KubeConfig)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrGetDevSpaceAppInfo, nil)
		return
	}
	if isBasespace {
		result := setupcluster.MeshDevInfo{
			Header: devspace.TraceHeader,
			Apps: meshManager.GetBaseDevSpaceAppInfo(
				&setupcluster.MeshDevInfo{
					BaseNamespace: devspace.Namespace},
			),
		}
		result.SortApps()
		api.SendResponse(c, nil, result)
		return
	}

	info := &setupcluster.MeshDevInfo{
		BaseNamespace:    basespace.Namespace,
		MeshDevNamespace: devspace.Namespace,
	}
	apps, err := meshManager.GetAPPInfo(info)
	if err != nil {
		log.Error(err)
		api.SendResponse(c, errno.ErrGetDevSpaceAppInfo, nil)
		return
	}
	result := setupcluster.MeshDevInfo{
		Header: devspace.TraceHeader,
		Apps:   apps,
	}
	result.SortApps()
	api.SendResponse(c, nil, result)
}
