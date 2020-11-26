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
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster_user"

	"github.com/pkg/errors"
)

type ClusterUserService interface {
	Create(ctx context.Context, applicationId, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace string) (model.ClusterUserModel, error)
	Delete(ctx context.Context, id uint64) error
	BatchDelete(ctx context.Context, ids []uint64) error
	GetFirst(ctx context.Context, models model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetList(ctx context.Context, models model.ClusterUserModel) ([]*model.ClusterUserModel, error)
	Update(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	Close()
}

type clusterUserService struct {
	clusterUserRepo cluster_user.ClusterUserRepo
}

func NewClusterUserService() ClusterUserService {
	db := model.GetDB()
	return &clusterUserService{clusterUserRepo: cluster_user.NewApplicationClusterRepo(db)}
}

func (srv *clusterUserService) BatchDelete(ctx context.Context, ids []uint64) error {
	return srv.clusterUserRepo.BatchDelete(ctx, ids)
}

func (srv *clusterUserService) Delete(ctx context.Context, id uint64) error {
	return srv.clusterUserRepo.Delete(ctx, id)
}

func (srv *clusterUserService) Update(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error) {
	_, err := srv.clusterUserRepo.Update(ctx, models)
	if err != nil {
		return models, err
	}
	return models, nil
}

func (srv *clusterUserService) GetList(ctx context.Context, models model.ClusterUserModel) ([]*model.ClusterUserModel, error) {
	result, err := srv.clusterUserRepo.GetList(ctx, models)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (srv *clusterUserService) GetFirst(ctx context.Context, models model.ClusterUserModel) (*model.ClusterUserModel, error) {
	result, err := srv.clusterUserRepo.GetFirst(ctx, models)
	if err != nil {
		return nil, errors.Wrapf(err, "GetFirst users_cluster error")
	}
	return result, nil
}

func (srv *clusterUserService) Create(ctx context.Context, applicationId, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace string) (model.ClusterUserModel, error) {
	c := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId,
		ClusterId:     clusterId,
		KubeConfig:    kubeConfig,
		Namespace:     devNameSpace,
	}
	result, err := srv.clusterUserRepo.Create(ctx, c)
	if err != nil {
		return result, errors.Wrapf(err, "create application_cluster")
	}
	return result, nil
}

func (srv *clusterUserService) Close() {
	srv.clusterUserRepo.Close()
}
