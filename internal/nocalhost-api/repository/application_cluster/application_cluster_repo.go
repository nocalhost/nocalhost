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

package application_cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ApplicationClusterRepo interface {
	Create(
		ctx context.Context,
		model model.ApplicationClusterModel,
	) (model.ApplicationClusterModel, error)
	GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error)
	GetList(ctx context.Context, id uint64) ([]*model.ApplicationClusterModel, error)
	GetJoinCluster(ctx context.Context, id uint64) ([]*model.ApplicationClusterJoinModel, error)
	Close()
}

type applicationClusterRepo struct {
	db *gorm.DB
}

func NewApplicationClusterRepo(db *gorm.DB) ApplicationClusterRepo {
	return &applicationClusterRepo{
		db: db,
	}
}

func (repo *applicationClusterRepo) GetJoinCluster(
	ctx context.Context,
	id uint64,
) ([]*model.ApplicationClusterJoinModel, error) {
	//TODO group by in mysql 5.7 require full select cols https://stackoverflow.com/questions/36207042/error-code-1055-incompatible-with-sql-mode-only-full-group-by
	var result []*model.ApplicationClusterJoinModel
	err := repo.db.Table("applications_clusters as ac").
		Select("count(ac.id) as dev_space_count,ac.cluster_id,ac.application_id,c.name as cluster_name,c.info as cluster_info,min(ac.created_at) as created_at,if(c.info is null,\"0\",\"1\") as cluster_status").
		Joins("left join clusters as c on c.id=ac.cluster_id").
		Joins("left join clusters_users as cu on cu.application_id=ac.application_id and cu.cluster_id=ac.cluster_id").
		Where("ac.application_id=?", id).
		Group("ac.cluster_id,ac.application_id,cluster_name,cluster_info,cluster_status").
		Scan(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *applicationClusterRepo) GetList(
	ctx context.Context,
	id uint64,
) ([]*model.ApplicationClusterModel, error) {
	var result []*model.ApplicationClusterModel
	err := repo.db.Where("application_id=?", id).Find(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *applicationClusterRepo) GetFirst(
	ctx context.Context,
	id uint64,
) (model.ApplicationClusterModel, error) {
	result := model.ApplicationClusterModel{}
	err := repo.db.First("applciation_id=?", id)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *applicationClusterRepo) Create(
	ctx context.Context,
	clusterModel model.ApplicationClusterModel,
) (model.ApplicationClusterModel, error) {
	err := repo.db.Create(&clusterModel).Error
	if err != nil {
		return clusterModel, errors.Wrap(
			err,
			"[application_cluster_repo] create application_cluster error",
		)
	}

	return clusterModel, nil
}

// Close close db
func (repo *applicationClusterRepo) Close() {
	repo.db.Close()
}
