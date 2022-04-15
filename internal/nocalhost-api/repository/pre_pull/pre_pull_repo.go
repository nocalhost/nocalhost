/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pre_pull

import (
	"context"
	"errors"
	"github.com/jinzhu/gorm"
	"nocalhost/internal/nocalhost-api/model"
)

type PrePullRepoRepoBase struct {
	db *gorm.DB
}

func NewPrePullRepoRepo(db *gorm.DB) *PrePullRepoRepoBase {
	return &PrePullRepoRepoBase{
		db: db,
	}
}

func (repo *PrePullRepoRepoBase) GetAll(ctx context.Context) ([]model.PrePullModel, error) {
	var images []model.PrePullModel
	result := repo.db.Find(&images)
	if result.RowsAffected > 0 {
		return images, nil
	}
	return nil, errors.New("record not found")
}

func (repo *PrePullRepoRepoBase) Close() {
	repo.db.Close()
}
