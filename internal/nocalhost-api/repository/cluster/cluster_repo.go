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
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ClusterRepo interface {
	Create(ctx context.Context, user model.ClusterModel) (id uint64, err error)
	Get(ctx context.Context, clusterId uint64, userId uint64) (model.ClusterModel, error)
	Close()
}

type clusterBaseRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) ClusterRepo {
	return &clusterBaseRepo{
		db: db,
	}
}

func (repo *clusterBaseRepo) Create(ctx context.Context, cluster model.ClusterModel) (id uint64, err error) {
	err = repo.db.Create(&cluster).Error
	if err != nil {
		return 0, errors.Wrap(err, "[cluster_repo] create user err")
	}

	return cluster.ID, nil
}

func (repo *clusterBaseRepo) Get(ctx context.Context, clusterId uint64, userId uint64) (model.ClusterModel, error) {
	cluster := model.ClusterModel{}
	if result := repo.db.Where("id=? and user_id=?", clusterId, userId).First(&cluster); result.Error != nil {
		log.Warnf("[cluster_repo] get cluster for user: %v id: %v error", userId, clusterId)
		return cluster, result.Error
	}
	return cluster, nil
}

// Close close db
func (repo *clusterBaseRepo) Close() {
	repo.db.Close()
}
