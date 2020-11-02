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

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ApplicationClusterRepo interface {
	Create(ctx context.Context, model model.ApplicationClusterModel) (uint64, error)
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

func (repo *applicationClusterRepo) Create(ctx context.Context, clusterModel model.ApplicationClusterModel) (id uint64, err error) {
	err = repo.db.Create(&clusterModel).Error
	if err != nil {
		return 0, errors.Wrap(err, "[application_cluster_repo] create application_cluster error")
	}

	return clusterModel.ID, nil
}

// Close close db
func (repo *applicationClusterRepo) Close() {
	repo.db.Close()
}
