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
	return srv.clusterUserRepo.UpdateKubeConfig(ctx, models)
}

func (srv *clusterUserService) GetJoinCluster(
	ctx context.Context, condition model.ClusterUserJoinCluster,
) ([]*model.ClusterUserJoinCluster, error) {
	return srv.clusterUserRepo.GetJoinCluster(ctx, condition)
}

func (srv *clusterUserService) DeleteByWhere(ctx context.Context, models model.ClusterUserModel) error {
	return srv.clusterUserRepo.DeleteByWhere(ctx, models)
}

func (srv *clusterUserService) BatchDelete(ctx context.Context, ids []uint64) error {
	return srv.clusterUserRepo.BatchDelete(ctx, ids)
}

func (srv *clusterUserService) Delete(ctx context.Context, id uint64) error {
	return srv.clusterUserRepo.Delete(ctx, id)
}

func (srv *clusterUserService) Update(ctx context.Context, models *model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	_, err := srv.clusterUserRepo.Update(ctx, models)
	if err != nil {
		return models, err
	}
	return models, nil
}

func (srv *clusterUserService) GetList(ctx context.Context, models model.ClusterUserModel) (
	[]*model.ClusterUserModel, error,
) {

	result, err := srv.clusterUserRepo.GetList(ctx, models)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (srv *clusterUserService) GetFirst(ctx context.Context, models model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	result, err := srv.clusterUserRepo.GetFirst(ctx, models)
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
	result, err := srv.clusterUserRepo.Create(ctx, c)
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
	result, err := srv.clusterUserRepo.Create(ctx, c)
	if err != nil {
		return result, errors.Wrapf(err, "create application_cluster")
	}
	return result, nil
}

func (srv *clusterUserService) GetJoinClusterAndAppAndUser(
	ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
) ([]*model.ClusterUserJoinClusterAndAppAndUser, error) {
	return srv.clusterUserRepo.GetJoinClusterAndAppAndUser(ctx, condition)
}

func (srv *clusterUserService) GetJoinClusterAndAppAndUserDetail(
	ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser,
) (*model.ClusterUserJoinClusterAndAppAndUser, error) {
	return srv.clusterUserRepo.GetJoinClusterAndAppAndUserDetail(ctx, condition)
}

func (srv *clusterUserService) ListByUser(ctx context.Context, userId uint64) ([]*model.ClusterUserPluginModel, error) {
	return srv.clusterUserRepo.ListByUser(ctx, userId)
}

func (srv *clusterUserService) Close() {
	srv.clusterUserRepo.Close()
}
