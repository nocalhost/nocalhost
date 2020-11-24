/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster_user

import (
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// Create 创建开发环境
// @Summary 创建开发环境
// @Description 为创建应用的开发环境
// @Tags 应用
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body cluster_user.ClusterUserCreateRequest true "cluster user info"
// @Param id path uint64 true "应用 ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/application/{id}/create_space [post]
func Create(c *gin.Context) {
	var req ClusterUserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind ApplicationCluster params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId := cast.ToUint64(req.UserId)
	webUserId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	// check application
	if _, err := service.Svc.ApplicationSvc().Get(c, applicationId, webUserId.(uint64)); err != nil {
		api.SendResponse(c, errno.ErrPermissionApplication, nil)
		return
	}
	// check cluster
	clusterData, err := service.Svc.ClusterSvc().Get(c, *req.ClusterId, webUserId.(uint64))
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}
	// check if has auth
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId,
	}
	record, hasRecord := service.Svc.ClusterUser().GetFirst(c, cu)
	if hasRecord == nil {
		log.Infof("cluster users %v", record)
		api.SendResponse(c, errno.ErrBindUserClsuterRepeat, nil)
		return
	}

	// create namespace
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewGoClient(KubeConfig)
	if err != nil {
		log.Errorf("client go got err %v", err)
		api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		return
	}
	// create cluster devs
	devNamespace := goClient.GenerateNsName(userId)
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)
	secret, err := clusterDevsSetUp.CreateNS(devNamespace, "").CreateServiceAccount("", devNamespace).CreateRole(global.NocalhostDevRoleName, devNamespace).CreateRoleBinding(global.NocalhostDevRoleBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevServiceAccountName).CreateRoleBinding(global.NocalhostDevRoleDefaultBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevDefaultServiceAccountName).GetServiceAccount(global.NocalhostDevServiceAccountName, devNamespace).GetServiceAccountSecret("", devNamespace)
	KubeConfigYaml, err, nerrno := setupcluster.NewDevKubeConfigReader(secret, clusterData.Server, devNamespace).GetCA().GetToken().AssembleDevKubeConfig().ToYamlString()
	if err != nil {
		api.SendResponse(c, nerrno, nil)
		return
	}

	result, err := service.Svc.ClusterUser().Create(c, applicationId, *req.ClusterId, userId, *req.Memory, *req.Cpu, KubeConfigYaml, devNamespace)
	if err != nil {
		log.Warnf("create ApplicationCluster err: %v", err)
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}

	api.SendResponse(c, nil, result)
}
