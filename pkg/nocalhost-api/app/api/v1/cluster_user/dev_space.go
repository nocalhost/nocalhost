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
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

type DevSpace struct {
	DevSpaceParams ClusterUserCreateRequest
	c              *gin.Context
	KubeConfig     []byte
}

func NewDevSpace(
	devSpaceParams ClusterUserCreateRequest,
	c *gin.Context,
	kubeConfig []byte,
) *DevSpace {
	return &DevSpace{
		DevSpaceParams: devSpaceParams,
		c:              c,
		KubeConfig:     kubeConfig,
	}
}

func (d *DevSpace) Delete() error {
	goClient, err := clientgo.NewAdminGoClient(d.KubeConfig)

	// get client go and check if is admin Kubeconfig
	if err != nil {
		switch err.(type) {
		case *errno.Errno:
			return err
		default:
			return errno.ErrClusterKubeErr
		}
	}

	_, _ = goClient.DeleteNS(d.DevSpaceParams.NameSpace)

	// delete database cluster-user dev space
	dErr := service.Svc.ClusterUser().Delete(d.c, *d.DevSpaceParams.ID)
	if dErr != nil {
		return errno.ErrDeletedClusterButDatabaseFail
	}
	return nil
}

func (d *DevSpace) Create() (*model.ClusterUserModel, error) {
	userId := cast.ToUint64(d.DevSpaceParams.UserId)
	clusterId := cast.ToUint64(d.DevSpaceParams.ClusterId)
	applicationId := cast.ToUint64(d.DevSpaceParams.ApplicationId)

	// get user
	usersRecord, err := service.Svc.UserSvc().GetUserByID(d.c, userId)
	if err != nil {
		return nil, errno.ErrUserNotFound
	}

	clusterRecord, err := service.Svc.ClusterSvc().Get(context.TODO(), clusterId)
	if err != nil {
		return nil, errno.ErrClusterNotFound
	}

	spaceName := clusterRecord.Name + "[" + usersRecord.Name + "]"
	if d.DevSpaceParams.SpaceName != "" {
		spaceName = d.DevSpaceParams.SpaceName
	}

	// check cluster
	clusterData, err := service.Svc.ClusterSvc().Get(d.c, *d.DevSpaceParams.ClusterId)
	if err != nil {
		return nil, errno.ErrPermissionCluster
	}

	if applicationId != 0 {

		// check if has created
		cu := model.ClusterUserModel{
			ApplicationId: applicationId,
			UserId:        userId,
		}
		_, hasRecord := service.Svc.ClusterUser().GetFirst(d.c, cu)

		// for adapt current version, prevent can't not create devSpace on same namespace
		if hasRecord == nil {
			return nil, errno.ErrBindUserApplicationRepeat
		}
	}

	// create namespace
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewAdminGoClient(KubeConfig)

	// get client go and check if is admin Kubeconfig
	if err != nil {
		switch err.(type) {
		case *errno.Errno:
			return nil, err
		default:
			return nil, errno.ErrClusterKubeErr
		}
	}
	// create cluster devs
	devNamespace := goClient.GenerateNsName(userId)
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)
	secret, err := clusterDevsSetUp.CreateNS(devNamespace, "").
		CreateServiceAccount("", devNamespace).
		CreateRole(global.NocalhostDevRoleName, devNamespace).
		CreateRoleBinding(global.NocalhostDevRoleBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevServiceAccountName).
		CreateRoleBinding(global.NocalhostDevRoleDefaultBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevDefaultServiceAccountName).
		GetServiceAccount(global.NocalhostDevServiceAccountName, devNamespace).
		GetServiceAccountSecret("", devNamespace)
	KubeConfigYaml, err, nerrno := setupcluster.NewDevKubeConfigReader(secret, clusterData.Server, devNamespace).
		GetCA().
		GetToken().
		AssembleDevKubeConfig().
		ToYamlString()
	if err != nil {
		return nil, nerrno
	}

	// create namespace ResouceQuota and container limitRange
	res := d.DevSpaceParams.SpaceResourceLimit
	if res == nil {
		res = &SpaceResourceLimit{}
	}

	res.ContainerEphemeralStorage = "1Gi"

	clusterDevsSetUp.CreateResouceQuota("rq-"+devNamespace, devNamespace, res.SpaceReqMem,
		res.SpaceReqCpu, res.SpaceLimitsMem, res.SpaceLimitsCpu, res.SpaceStorageCapacity, res.SpaceEphemeralStorage,
		res.SpacePvcCount, res.SpaceLbCount).
		CreateLimitRange("lr-"+devNamespace, devNamespace,
			res.ContainerReqMem, res.ContainerLimitsMem, res.ContainerReqCpu, res.ContainerLimitsCpu, res.ContainerEphemeralStorage)

	resString, err := json.Marshal(res)
	result, err := service.Svc.ClusterUser().
		Create(d.c, applicationId, *d.DevSpaceParams.ClusterId, userId, *d.DevSpaceParams.Memory, *d.DevSpaceParams.Cpu, KubeConfigYaml, devNamespace, spaceName, string(resString))
	if err != nil {
		return nil, errno.ErrBindApplicationClsuter
	}

	_ = service.Svc.ApplicationUser().BatchInsert(d.c, applicationId, []uint64{userId})
	return &result, nil
}
