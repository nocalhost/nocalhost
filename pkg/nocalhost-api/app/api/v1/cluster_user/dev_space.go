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

func NewDevSpace(devSpaceParams ClusterUserCreateRequest, c *gin.Context, kubeConfig []byte) *DevSpace {
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

	// get user
	usersRecord, err := service.Svc.UserSvc().GetUserByID(d.c, userId)
	if err != nil {
		return nil, errno.ErrUserNotFound
	}

	// check cluster
	clusterRecord, err := service.Svc.ClusterSvc().Get(context.TODO(), clusterId)
	if err != nil {
		return nil, errno.ErrClusterNotFound
	}

	if d.DevSpaceParams.ClusterAdmin == nil || *d.DevSpaceParams.ClusterAdmin == 0 {
		return d.createDevSpace(clusterRecord, usersRecord)
	} else {
		return d.createClusterDevSpace(clusterRecord, usersRecord)
	}
}

func (d *DevSpace) createClusterDevSpace(
	clusterRecord model.ClusterModel, usersRecord *model.UserBaseModel,
) (*model.ClusterUserModel, error) {
	trueFlag := uint64(1)
	list, err := service.Svc.ClusterUser().GetList(
		context.TODO(), model.ClusterUserModel{
			ClusterId: clusterRecord.ID,
			UserId: usersRecord.ID,
			ClusterAdmin: &trueFlag,
		},
	)
	if len(list) > 0 {
		return nil, errno.ErrAlreadyExist
	}

	result, err := service.Svc.ClusterUser().CreateClusterAdminSpace(
		context.TODO(), clusterRecord.ID, usersRecord.ID, d.DevSpaceParams.SpaceName,
	)
	if err != nil {
		return nil, errno.ErrBindApplicationClsuter
	}

	if err := service.Svc.AuthorizeClusterToUser(clusterRecord.ID, usersRecord.ID); err != nil {
		return nil, err
	}

	return &result, nil
}

func (d *DevSpace) createDevSpace(
	clusterRecord model.ClusterModel, usersRecord *model.UserBaseModel,
) (*model.ClusterUserModel, error) {

	applicationId := cast.ToUint64(d.DevSpaceParams.ApplicationId)

	spaceName := clusterRecord.Name + "[" + usersRecord.Name + "]"
	if d.DevSpaceParams.SpaceName != "" {
		spaceName = d.DevSpaceParams.SpaceName
	}

	// create namespace
	var KubeConfig = []byte(clusterRecord.KubeConfig)
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
	devNamespace := goClient.GenerateNsName(usersRecord.ID)
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)
	secret, err := clusterDevsSetUp.
		CreateNS(devNamespace, "").
		CreateServiceAccount("", devNamespace).
		CreateRole(global.NocalhostDevRoleName, devNamespace).
		CreateRoleBinding(
		global.NocalhostDevRoleBindingName, devNamespace, global.NocalhostDevRoleName,
		global.NocalhostDevServiceAccountName,
	).
		CreateRoleBinding(
		global.NocalhostDevRoleDefaultBindingName, devNamespace, global.NocalhostDevRoleName,
		global.NocalhostDevDefaultServiceAccountName,
		).
		GetServiceAccount(global.NocalhostDevServiceAccountName, devNamespace).
		GetServiceAccountSecret("", devNamespace)

	KubeConfigYaml, err, nerrno := setupcluster.
		NewDevKubeConfigReader(secret, clusterRecord.Server, devNamespace).
		GetCA().
		GetToken().
		AssembleDevKubeConfig().ToYamlString()
	if err != nil {
		return nil, nerrno
	}

	// create namespace ResouceQuota and container limitRange
	res := d.DevSpaceParams.SpaceResourceLimit
	if res == nil {
		res = &SpaceResourceLimit{}
	}

	res.ContainerEphemeralStorage = "1Gi"

	clusterDevsSetUp.CreateResourceQuota(
		"rq-"+devNamespace, devNamespace, res.SpaceReqMem,
		res.SpaceReqCpu, res.SpaceLimitsMem, res.SpaceLimitsCpu, res.SpaceStorageCapacity, res.SpaceEphemeralStorage,
		res.SpacePvcCount, res.SpaceLbCount,
	).CreateLimitRange(
		"lr-"+devNamespace, devNamespace,
		res.ContainerReqMem, res.ContainerLimitsMem, res.ContainerReqCpu, res.ContainerLimitsCpu,
		res.ContainerEphemeralStorage,
	)

	resString, err := json.Marshal(res)
	result, err := service.Svc.ClusterUser().Create(
		d.c, *d.DevSpaceParams.ClusterId, usersRecord.ID, *d.DevSpaceParams.Memory, *d.DevSpaceParams.Cpu,
		KubeConfigYaml, devNamespace, spaceName, string(resString),
	)
	if err != nil {
		return nil, errno.ErrBindApplicationClsuter
	}

	// auth application to user
	_ = service.Svc.ApplicationUser().BatchInsert(d.c, applicationId, []uint64{usersRecord.ID})

	// authorize namespace to user
	if err := service.Svc.AuthorizeNsToUser(clusterRecord.ID, usersRecord.ID, result.Namespace); err != nil {
		return nil, err
	}

	return &result, nil
}
