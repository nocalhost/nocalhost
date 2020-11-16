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

package cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster"

	"github.com/pkg/errors"
)

type ClusterService interface {
	Create(ctx context.Context, name, marks, kubeconfig, server, clusterInfo string, userId uint64) error
	Get(ctx context.Context, id, userId uint64) (model.ClusterModel, error)
	GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error)
	GetList(ctx context.Context) ([]*model.ClusterList, error)
	Close()
}

type clusterService struct {
	clusterRepo cluster.ClusterRepo
}

func NewClusterService() ClusterService {
	db := model.GetDB()
	return &clusterService{
		clusterRepo: cluster.NewClusterRepo(db),
	}
}

func (srv *clusterService) GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error) {
	return srv.clusterRepo.GetAny(ctx, where)
}

func (srv *clusterService) Create(ctx context.Context, name, marks, kubeconfig, server, clusterInfo string, userId uint64) error {
	c := model.ClusterModel{
		Name:       name,
		Marks:      marks,
		UserId:     userId,
		Server:     server,
		KubeConfig: kubeconfig,
		Info:       clusterInfo,
	}
	_, err := srv.clusterRepo.Create(ctx, c)
	if err != nil {
		return errors.Wrapf(err, "create cluster")
	}
	return nil
}

func (srv *clusterService) GetList(ctx context.Context) ([]*model.ClusterList, error) {
	result, _ := srv.clusterRepo.GetList(ctx)
	return result, nil
}

func (srv *clusterService) Get(ctx context.Context, id, userId uint64) (model.ClusterModel, error) {
	result, err := srv.clusterRepo.Get(ctx, id, userId)
	if err != nil {
		return result, errors.Wrapf(err, "get cluster")
	}
	return result, nil
}

func (srv *clusterService) Close() {
	srv.clusterRepo.Close()
}
