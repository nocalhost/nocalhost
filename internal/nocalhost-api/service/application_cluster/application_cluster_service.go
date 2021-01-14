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

package application_cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_cluster"

	"github.com/pkg/errors"
)

type ApplicationClusterService interface {
	Create(ctx context.Context, applicationId uint64, clusterId uint64) (model.ApplicationClusterModel, error)
	GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error)
	GetList(ctx context.Context, id uint64) ([]*model.ApplicationClusterModel, error)
	GetJoinCluster(ctx context.Context, id uint64) ([]*model.ApplicationClusterJoinModel, error)
	Close()
}

type applicationClusterService struct {
	applicationClusterRepo application_cluster.ApplicationClusterRepo
}

func NewApplicationClusterService() ApplicationClusterService {
	db := model.GetDB()
	return &applicationClusterService{applicationClusterRepo: application_cluster.NewApplicationClusterRepo(db)}
}

func (srv *applicationClusterService) GetJoinCluster(ctx context.Context, id uint64) ([]*model.ApplicationClusterJoinModel, error) {
	return srv.applicationClusterRepo.GetJoinCluster(ctx, id)
}

func (srv *applicationClusterService) GetList(ctx context.Context, id uint64) ([]*model.ApplicationClusterModel, error) {
	return srv.applicationClusterRepo.GetList(ctx, id)
}

func (srv *applicationClusterService) GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error) {
	return srv.applicationClusterRepo.GetFirst(ctx, id)
}

func (srv *applicationClusterService) Create(ctx context.Context, applicationId uint64, clusterId uint64) (model.ApplicationClusterModel, error) {
	c := model.ApplicationClusterModel{
		ApplicationId: applicationId,
		ClusterId:     clusterId,
	}
	_, err := srv.applicationClusterRepo.Create(ctx, c)
	if err != nil {
		return c, errors.Wrapf(err, "create application_cluster")
	}
	return c, nil
}

func (srv *applicationClusterService) Close() {
	srv.applicationClusterRepo.Close()
}
