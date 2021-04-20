/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster_user

import (
	"context"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/errno"

	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
)

// @Summary Plug-in Get personal application development environment (kubeconfig) (obsolete)
// @Description Get personal application development environment (kubeconfig) (obsolete)
// @Tags Plug-in
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Application ID"
// @Success 200 {object} model.ClusterUserModel "Application development environment parameters, including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/application/{id}/dev_space [get]
func GetFirst(c *gin.Context) {
	userId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId.(uint64),
	}
	result, err := service.Svc.ClusterUser().GetFirst(c, cu)
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
// @Success 200 {object} model.ClusterUserModel "Application development environment parameters, including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/application/{id}/dev_space_list [get]
func GetList(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
	}
	result, err := service.Svc.ClusterUser().GetList(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

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

	result, err := service.Svc.ClusterUser().GetList(c, cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// list user's dev space distinct by user id
func ListByUserId(c *gin.Context) {
	userId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ClusterUser().ListByUser(c, userId)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	list, err := service.Svc.ClusterSvc().GetList(context.TODO())
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
// @Success 200 {object} model.ClusterUserModel "Application development environment parameters, including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/application/{id}/dev_space/{space_id}/detail [get]
func GetDevSpaceDetail(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))
	spaceId := cast.ToUint64(c.Param("space_id"))
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		ID:            spaceId,
	}
	result, err := service.Svc.ClusterUser().GetFirst(c, cu)
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
// @Success 200 {object} model.ClusterUserJoinClusterAndAppAndUser "Application development environment parameters, including kubeconfig, status=0 application not installed, 1 installed"
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

	result, err := service.Svc.ClusterUser().GetJoinClusterAndAppAndUser(c, condition)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get the details of a development environment of the application
// @Description Get dev space detail from application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "DevSpace ID"
// @Success 200 {object} model.ClusterUserJoinClusterAndAppAndUser "Application development environment parameters, including kubeconfig, status=0 application not installed, 1 installed"
// @Router /v1/dev_space/{id}/detail [get]
func GetJoinClusterAndAppAndUserDetail(c *gin.Context) {
	condition := model.ClusterUserJoinClusterAndAppAndUser{
		ID: cast.ToUint64(c.Param("id")),
	}

	if !ginbase.IsAdmin(c) {
		user, _ := ginbase.LoginUser(c)
		condition.UserId = user
	}

	result, err := service.Svc.ClusterUser().GetJoinClusterAndAppAndUserDetail(c, condition)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
