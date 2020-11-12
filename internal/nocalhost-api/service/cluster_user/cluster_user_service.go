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
	Create(ctx context.Context, applicationId, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace string) error
	GetFirst(ctx context.Context, models model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetList(ctx context.Context, c map[string]interface{}) ([]*model.ClusterUserModel, error)
	Update(ctx context.Context, models *model.ClusterUserModel) error
	Close()
}

type clusterUserService struct {
	clusterUserRepo cluster_user.ClusterUserRepo
}

func NewClusterUserService() ClusterUserService {
	db := model.GetDB()
	return &clusterUserService{clusterUserRepo: cluster_user.NewApplicationClusterRepo(db)}
}

func (srv *clusterUserService) Update(ctx context.Context, models *model.ClusterUserModel) error {
	err := srv.clusterUserRepo.Update(ctx, models)
	if err != nil {
		return err
	}
	return nil
}

func (srv *clusterUserService) GetList(ctx context.Context, c map[string]interface{}) ([]*model.ClusterUserModel, error) {
	result, err := srv.clusterUserRepo.GetList(ctx, c)
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

func (srv *clusterUserService) Create(ctx context.Context, applicationId, clusterId, userId, memory, cpu uint64, kubeConfig, devNameSpace string) error {
	c := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId,
		ClusterId:     clusterId,
		KubeConfig:    kubeConfig,
		Namespace:     devNameSpace,
	}
	_, err := srv.clusterUserRepo.Create(ctx, c)
	if err != nil {
		return errors.Wrapf(err, "create application_cluster")
	}
	return nil
}

func (srv *clusterUserService) Close() {
	srv.clusterUserRepo.Close()
}
