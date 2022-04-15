/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/cache"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster_user"
)

type ClusterUser struct {
	clusterUserRepo *cluster_user.ClusterUserRepoBase
}

func NewClusterUserService() *ClusterUser {
	db := model.GetDB()
	return &ClusterUser{clusterUserRepo: cluster_user.NewApplicationClusterRepo(db)}
}

func (srv *ClusterUser) Evict(id uint64) {
	c := cache.Module(cache.CLUSTER_USER)
	value, err := c.Value(id)
	if err == nil {
		cu := value.Data().(*model.ClusterUserModel)
		_, _ = c.Delete(keyForClusterAndNameSpace(cu.ClusterId, cu.Namespace))
		_, _ = c.Delete(id)
	}
	_, _ = c.Delete("*")
}

func (srv *ClusterUser) GetAllCache() []model.ClusterUserModel {
	c := cache.Module(cache.CLUSTER_USER)
	value, err := c.Value("*")

	resultList := []model.ClusterUserModel{}
	if err == nil {
		clusterUserModels := value.Data().([]*model.ClusterUserModel)
		for _, userModel := range clusterUserModels {
			resultList = append(resultList, *userModel)
		}
		return resultList
	}

	list, err := srv.clusterUserRepo.GetList(model.ClusterUserModel{})
	if err != nil {
		return resultList
	}

	c.Add("*", cache.OUT_OF_DATE, list)
	for _, result := range list {
		c.Add(keyForClusterAndNameSpace(result.ClusterId, result.Namespace), cache.OUT_OF_DATE, result)
		c.Add(result.ID, cache.OUT_OF_DATE, result)
		resultList = append(resultList, *result)
	}
	return resultList
}

func (srv *ClusterUser) GetCache(id uint64) (
	model.ClusterUserModel, error,
) {
	c := cache.Module(cache.CLUSTER_USER)
	value, err := c.Value(id)
	if err == nil {
		clusterUserModel := value.Data().(*model.ClusterUserModel)
		return *clusterUserModel, nil
	}

	result, err := srv.clusterUserRepo.GetFirst(
		model.ClusterUserModel{ID: id},
	)
	if err != nil {
		return model.ClusterUserModel{}, errors.Wrapf(err, "GetCache users_cluster error")
	}

	c.Add(keyForClusterAndNameSpace(result.ClusterId, result.Namespace), cache.OUT_OF_DATE, result)
	c.Add(result.ID, cache.OUT_OF_DATE, result)
	return *result, nil
}

func (srv *ClusterUser) GetCacheByClusterAndNameSpace(clusterId uint64, namespace string) (
	model.ClusterUserModel, error,
) {
	c := cache.Module(cache.CLUSTER_USER)
	value, err := c.Value(keyForClusterAndNameSpace(clusterId, namespace))
	if err == nil {
		clusterUserModel := value.Data().(*model.ClusterUserModel)
		return *clusterUserModel, nil
	}

	result, err := srv.clusterUserRepo.GetFirst(
		model.ClusterUserModel{ClusterId: clusterId, Namespace: namespace},
	)
	if err != nil {
		return model.ClusterUserModel{}, errors.Wrapf(err, "GetCache users_cluster error")
	}

	c.Add(keyForClusterAndNameSpace(result.ClusterId, result.Namespace), cache.OUT_OF_DATE, result)
	c.Add(result.ID, cache.OUT_OF_DATE, result)
	return *result, nil
}

func keyForClusterAndNameSpace(clusterId uint64, namespace string) string {
	return fmt.Sprintf("A:%v-%v", clusterId, namespace)
}

func (srv *ClusterUser) ListV2(models model.ClusterUserModel) (
	[]*model.ClusterUserV2, error,
) {

	list, err := srv.clusterUserRepo.ListWithFuzzySpaceName(models)
	if err != nil {
		return nil, err
	}

	var result []*model.ClusterUserV2
	for _, userModel := range list {
		item := &model.ClusterUserV2{}
		item.ID = userModel.ID
		item.UserId = userModel.UserId
		item.ClusterAdmin = userModel.ClusterAdmin
		item.Namespace = userModel.Namespace
		item.SpaceName = userModel.SpaceName
		item.ClusterId = userModel.ClusterId
		item.SpaceResourceLimit = userModel.SpaceResourceLimit
		item.CreatedAt = userModel.CreatedAt
		item.BaseDevSpaceId = userModel.BaseDevSpaceId
		item.IsBaseSpace = userModel.IsBaseSpace
		item.TraceHeader = userModel.TraceHeader
		result = append(result, item)
	}
	return result, nil
}

func (srv *ClusterUser) UpdateKubeConfig(
	ctx context.Context, models *model.ClusterUserModel,
) (*model.ClusterUserModel, error) {
	return srv.clusterUserRepo.UpdateKubeConfig(models)
}

func (srv *ClusterUser) GetJoinCluster(
	ctx context.Context, condition model.ClusterUserJoinCluster,
) ([]*model.ClusterUserJoinCluster, error) {
	return srv.clusterUserRepo.GetJoinCluster(condition)
}

func (srv *ClusterUser) BatchDelete(ctx context.Context, ids []uint64) error {
	defer func() {
		for _, id := range ids {
			srv.Evict(id)
		}
	}()
	return srv.clusterUserRepo.BatchDelete(ids)
}

func (srv *ClusterUser) Delete(ctx context.Context, id uint64) error {
	defer srv.Evict(id)
	return srv.clusterUserRepo.Delete(id)
}

func (srv *ClusterUser) Update(ctx context.Context, models *model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	result, err := srv.clusterUserRepo.Update(models)
	if err != nil {
		return models, err
	}

	srv.Evict(result.ID)
	return models, nil
}

func (srv *ClusterUser) GetList(ctx context.Context, models model.ClusterUserModel) (
	[]*model.ClusterUserModel, error,
) {

	result, err := srv.clusterUserRepo.GetList(models)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (srv *ClusterUser) GetFirst(ctx context.Context, models model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	result, err := srv.clusterUserRepo.GetFirst(models)
	if err != nil {
		return nil, errors.Wrapf(err, "GetFirst users_cluster error")
	}
	return result, nil
}

func (srv *ClusterUser) Create(
	ctx context.Context, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace, spaceName string,
	spaceResourceLimit string, isBaseName bool, protected bool,
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
		IsBaseSpace:        isBaseName,
		Protected:          protected,
	}
	result, err := srv.clusterUserRepo.Create(c)
	if err != nil {
		return result, errors.Wrapf(err, "create application_cluster")
	}
	srv.Evict(result.ID)
	return result, nil
}

func (srv *ClusterUser) CreateClusterAdminSpace(
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
	srv.Evict(result.ID)
	return result, nil
}

func (srv *ClusterUser) GetJoinClusterAndAppAndUser(
	ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
) ([]*model.ClusterUserJoinClusterAndAppAndUser, error) {
	return srv.clusterUserRepo.GetJoinClusterAndAppAndUser(condition)
}

func (srv *ClusterUser) GetJoinClusterAndAppAndUserDetail(
	ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
) (*model.ClusterUserJoinClusterAndAppAndUser, error) {
	return srv.clusterUserRepo.GetJoinClusterAndAppAndUserDetail(condition)
}

func (srv *ClusterUser) ListByUser(ctx context.Context, userId uint64) ([]*model.ClusterUserPluginModel, error) {
	return srv.clusterUserRepo.ListByUser(userId)
}

func (srv *ClusterUser) Close() {
	srv.clusterUserRepo.Close()
}
