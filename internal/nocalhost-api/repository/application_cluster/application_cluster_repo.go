/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package application_cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ApplicationClusterRepoBase struct {
	db *gorm.DB
}

func NewApplicationClusterRepo(db *gorm.DB) *ApplicationClusterRepoBase {
	return &ApplicationClusterRepoBase{
		db: db,
	}
}

func (repo *ApplicationClusterRepoBase) GetJoinCluster(
	ctx context.Context, id uint64,
) ([]*model.ApplicationClusterJoinModel, error) {
	// TODO group by in mysql 5.7 require full select cols
	// https://stackoverflow.com/questions/36207042/error-code-1055-incompatible-with-sql-mode-only-full-group-by
	var result []*model.ApplicationClusterJoinModel
	err := repo.db.Table("applications_clusters as ac").
		Select(
			"count(ac.id) as dev_space_count,ac.cluster_id,ac.application_id,c.name as cluster_name,"+
				"c.info as "+
				"cluster_info,min(ac.created_at) as created_at,if(c.info is null,\"0\",\"1\")"+
				" as cluster_status",
		).Joins("left join clusters as c on c.id=ac.cluster_id").Joins(
		"left join clusters_users as cu "+
			"on cu.application_id=ac.application_id"+
			" and cu.cluster_id=ac.cluster_id",
	).Where(
		"ac.application_id=?", id,
	).Group("ac.cluster_id,ac.application_id,cluster_name,cluster_info,cluster_status").Scan(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *ApplicationClusterRepoBase) GetList(ctx context.Context, id uint64) ([]*model.ApplicationClusterModel, error) {
	var result []*model.ApplicationClusterModel
	err := repo.db.Where("application_id=?", id).Find(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *ApplicationClusterRepoBase) GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error) {
	result := model.ApplicationClusterModel{}
	err := repo.db.First("applciation_id=?", id)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *ApplicationClusterRepoBase) Create(
	ctx context.Context, clusterModel model.ApplicationClusterModel,
) (model.ApplicationClusterModel, error) {
	err := repo.db.Create(&clusterModel).Error
	if err != nil {
		return clusterModel, errors.Wrap(err, "[application_cluster_repo] create application_cluster error")
	}

	return clusterModel, nil
}

// Close close db
func (repo *ApplicationClusterRepoBase) Close() {
	repo.db.Close()
}
