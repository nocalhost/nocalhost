/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ClusterBaseRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) *ClusterBaseRepo {
	return &ClusterBaseRepo{
		db: db,
	}
}

func (repo *ClusterBaseRepo) Update(
	ctx context.Context, update map[string]interface{}, clusterId uint64,
) (*model.ClusterModel, error) {
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

func (repo *ClusterBaseRepo) Delete(ctx context.Context, clusterId uint64) error {
	result := repo.db.Unscoped().Delete(&model.ClusterModel{}, clusterId)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

func (repo *ClusterBaseRepo) DeleteByCreator(ctx context.Context, userId uint64) error {
	result := repo.db.Exec("delete from clusters where user_id = ? and deleted_at is null", userId)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

func (repo *ClusterBaseRepo) GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error) {
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

func (repo *ClusterBaseRepo) GetList(ctx context.Context) ([]*model.ClusterList, error) {
	var result []*model.ClusterList
	repo.db.Raw(
		"select c.id,c.kubeconfig,c.name,c.server,c.extra_api_server,c.storage_class,c.info,c.user_id,c.created_at,count" +
			"(distinct cu.id) as users_count from clusters as c left join clusters_users as cu on c.id=cu.cluster_id" +
			" where c.deleted_at is null and cu.deleted_at is null group by c.id",
	).
		Scan(&result)
	return result, nil
}

func (repo *ClusterBaseRepo) Create(ctx context.Context, cluster model.ClusterModel) (model.ClusterModel, error) {
	err := repo.db.Create(&cluster).Error
	if err != nil {
		return cluster, errors.Wrap(err, "[cluster_repo] create user err")
	}

	return cluster, nil
}

func (repo *ClusterBaseRepo) Get(ctx context.Context, clusterId uint64) (model.ClusterModel, error) {
	cluster := model.ClusterModel{}
	if result := repo.db.Where("id=?", clusterId).First(&cluster); result.Error != nil {
		log.Warnf("[cluster_repo] get cluster for id: %v error", clusterId)
		return cluster, result.Error
	}
	return cluster, nil
}

// Close close db
func (repo *ClusterBaseRepo) Close() {
	repo.db.Close()
}
