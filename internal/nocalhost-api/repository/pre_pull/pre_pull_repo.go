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

package pre_pull

import (
	"context"
	"errors"
	"github.com/jinzhu/gorm"
	"nocalhost/internal/nocalhost-api/model"
)

type PrePullRepo interface {
	GetAll(ctx context.Context) ([]model.PrePullModel, error)
	Close()
}

type prePullRepoRepo struct {
	db *gorm.DB
}

func NewPrePullRepoRepo(db *gorm.DB) PrePullRepo {
	return &prePullRepoRepo{
		db: db,
	}
}

func (repo *prePullRepoRepo) GetAll(ctx context.Context) ([]model.PrePullModel, error) {
	var images []model.PrePullModel
	result := repo.db.Find(&images)
	if result.RowsAffected > 0 {
		return images, nil
	}
	return nil, errors.New("record not found")
}

func (repo *prePullRepoRepo) Close() {
	repo.db.Close()
}
