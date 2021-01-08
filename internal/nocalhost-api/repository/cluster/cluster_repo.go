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
	Create(ctx context.Context, user model.ClusterModel) (model.ClusterModel, error)
	Get(ctx context.Context, clusterId uint64) (model.ClusterModel, error)
	Delete(ctx context.Context, clusterId uint64) error
	GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error)
	Update(ctx context.Context, update map[string]interface{}, clusterId uint64) (*model.ClusterModel, error)
	GetList(ctx context.Context) ([]*model.ClusterList, error)
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

func (repo *clusterBaseRepo) Update(ctx context.Context, update map[string]interface{}, clusterId uint64) (*model.ClusterModel, error) {
	clusterModel := model.ClusterModel{}
	clusterResult := repo.db.Where("id = ?", clusterId).First(&clusterModel)
	if clusterResult.Error != nil {
		return &clusterModel, clusterResult.Error
	}
	result := repo.db.Model(&clusterModel).Update(update)
	if result.RowsAffected > 0 {
		return &clusterModel, nil
	}
	return &clusterModel, result.Error
}

func (repo *clusterBaseRepo) Delete(ctx context.Context, clusterId uint64) error {
	result := repo.db.Unscoped().Delete(&model.ClusterModel{}, clusterId)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

func (repo *clusterBaseRepo) GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error) {
	cluster := make([]*model.ClusterModel, 0)
	result := repo.db.Where(where).Find(&cluster)
	if result.Error != nil {
		return cluster, result.Error
	}
	if len(cluster) == 0 {
		return cluster, errors.New("cluster not found")
	}
	return cluster, nil
}

func (repo *clusterBaseRepo) GetList(ctx context.Context) ([]*model.ClusterList, error) {
	var result []*model.ClusterList
	repo.db.Raw("select c.id,c.kubeconfig,c.name,c.storage_class,c.info,c.user_id,c.created_at,count(distinct cu.id) as users_count from clusters as c left join clusters_users as cu on c.id=cu.cluster_id where c.deleted_at is null and cu.deleted_at is null group by c.id").Scan(&result)
	return result, nil
}

func (repo *clusterBaseRepo) Create(ctx context.Context, cluster model.ClusterModel) (model.ClusterModel, error) {
	err := repo.db.Create(&cluster).Error
	if err != nil {
		return cluster, errors.Wrap(err, "[cluster_repo] create user err")
	}

	return cluster, nil
}

func (repo *clusterBaseRepo) Get(ctx context.Context, clusterId uint64) (model.ClusterModel, error) {
	cluster := model.ClusterModel{}
	if result := repo.db.Where("id=?", clusterId).First(&cluster); result.Error != nil {
		log.Warnf("[cluster_repo] get cluster for id: %v error", clusterId)
		return cluster, result.Error
	}
	return cluster, nil
}

// Close close db
func (repo *clusterBaseRepo) Close() {
	repo.db.Close()
}
