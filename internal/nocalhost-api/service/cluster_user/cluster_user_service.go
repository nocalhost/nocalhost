/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cluster_user

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster_user"

	"github.com/pkg/errors"
)

type ClusterUserService interface {
	Create(
		ctx context.Context, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace, spaceName string,
		spaceResourceLimit string,
	) (model.ClusterUserModel, error)
	CreateClusterAdminSpace(ctx context.Context, clusterId, userId uint64, spaceName string) (
		model.ClusterUserModel, error,
	)
	Delete(ctx context.Context, id uint64) error
	DeleteByWhere(ctx context.Context, models model.ClusterUserModel) error
	BatchDelete(ctx context.Context, ids []uint64) error
	GetFirst(ctx context.Context, models model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetList(ctx context.Context, models model.ClusterUserModel) ([]*model.ClusterUserModel, error)
	GetJoinCluster(ctx context.Context, condition model.ClusterUserJoinCluster) ([]*model.ClusterUserJoinCluster, error)
	Update(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	UpdateKubeConfig(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetJoinClusterAndAppAndUser(
		ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
	) ([]*model.ClusterUserJoinClusterAndAppAndUser, error)
	GetJoinClusterAndAppAndUserDetail(
		ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
	) (*model.ClusterUserJoinClusterAndAppAndUser, error)
	ListByUser(ctx context.Context, userId uint64) ([]*model.ClusterUserPluginModel, error)
	Close()
}

type clusterUserService struct {
	clusterUserRepo cluster_user.ClusterUserRepo
}

func NewClusterUserService() ClusterUserService {
	db := model.GetDB()
	return &clusterUserService{clusterUserRepo: cluster_user.NewApplicationClusterRepo(db)}
}

func (srv *clusterUserService) UpdateKubeConfig(
	ctx context.Context, models *model.ClusterUserModel,
) (*model.ClusterUserModel, error) {
	return srv.clusterUserRepo.UpdateKubeConfig(models)
}

func (srv *clusterUserService) GetJoinCluster(
	ctx context.Context, condition model.ClusterUserJoinCluster,
) ([]*model.ClusterUserJoinCluster, error) {
	return srv.clusterUserRepo.GetJoinCluster(condition)
}

func (srv *clusterUserService) DeleteByWhere(ctx context.Context, models model.ClusterUserModel) error {
	return srv.clusterUserRepo.DeleteByWhere(models)
}

func (srv *clusterUserService) BatchDelete(ctx context.Context, ids []uint64) error {
	return srv.clusterUserRepo.BatchDelete(ids)
}

func (srv *clusterUserService) Delete(ctx context.Context, id uint64) error {
	return srv.clusterUserRepo.Delete(id)
}

func (srv *clusterUserService) Update(ctx context.Context, models *model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	_, err := srv.clusterUserRepo.Update(models)
	if err != nil {
		return models, err
	}
	return models, nil
}

func (srv *clusterUserService) GetList(ctx context.Context, models model.ClusterUserModel) (
	[]*model.ClusterUserModel, error,
) {

	result, err := srv.clusterUserRepo.GetList(models)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (srv *clusterUserService) GetFirst(ctx context.Context, models model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	result, err := srv.clusterUserRepo.GetFirst(models)
	if err != nil {
		return nil, errors.Wrapf(err, "GetFirst users_cluster error")
	}
	return result, nil
}

func (srv *clusterUserService) Create(
	ctx context.Context, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace, spaceName string,
	spaceResourceLimit string,
) (model.ClusterUserModel, error) {
	c := model.ClusterUserModel{

		// Deprecated
		ApplicationId:      0,
		UserId:             userId,
		ClusterId:          clusterId,
		KubeConfig:         kubeConfig,
		Namespace:          devNameSpace,
		SpaceName:          spaceName,
		SpaceResourceLimit: spaceResourceLimit,
	}
	result, err := srv.clusterUserRepo.Create(c)
	if err != nil {
		return result, errors.Wrapf(err, "create application_cluster")
	}
	return result, nil
}

func (srv *clusterUserService) CreateClusterAdminSpace(
	ctx context.Context, clusterId, userId uint64, spaceName string,
) (model.ClusterUserModel, error) {
	trueFlag := uint64(1)

	c := model.ClusterUserModel{
		SpaceName:    spaceName,
		ClusterId:    clusterId,
		UserId:       userId,
		Namespace:    "*",
		ClusterAdmin: &trueFlag,
	}
	result, err := srv.clusterUserRepo.Create(c)
	if err != nil {
		return result, errors.Wrapf(err, "create application_cluster")
	}
	return result, nil
}

func (srv *clusterUserService) GetJoinClusterAndAppAndUser(
	ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
) ([]*model.ClusterUserJoinClusterAndAppAndUser, error) {
	return srv.clusterUserRepo.GetJoinClusterAndAppAndUser(condition)
}

func (srv *clusterUserService) GetJoinClusterAndAppAndUserDetail(
	ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
) (*model.ClusterUserJoinClusterAndAppAndUser, error) {
	return srv.clusterUserRepo.GetJoinClusterAndAppAndUserDetail(condition)
}

func (srv *clusterUserService) ListByUser(ctx context.Context, userId uint64) ([]*model.ClusterUserPluginModel, error) {
	return srv.clusterUserRepo.ListByUser(userId)
}

func (srv *clusterUserService) Close() {
	srv.clusterUserRepo.Close()
}
