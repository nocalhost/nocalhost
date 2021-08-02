/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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

	if d.DevSpaceParams.SpaceName == "" {
		d.DevSpaceParams.SpaceName = clusterRecord.Name + "[" + usersRecord.Name + "]"
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
			ClusterId:    clusterRecord.ID,
			UserId:       usersRecord.ID,
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
		KubeConfigYaml, devNamespace, d.DevSpaceParams.SpaceName, string(resString),
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
