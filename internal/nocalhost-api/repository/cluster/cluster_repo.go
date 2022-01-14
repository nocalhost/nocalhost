/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"time"
)

type ClusterRepo interface {
	Create(ctx context.Context, user model.ClusterModel) (model.ClusterModel, error)
	Get(ctx context.Context, clusterId uint64) (model.ClusterModel, error)
	Delete(ctx context.Context, clusterId uint64) error
	DeleteByCreator(ctx context.Context, clusterId uint64) error
	GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error)
	Update(ctx context.Context, update map[string]interface{}, clusterId uint64) (*model.ClusterModel, error)
	GetList(ctx context.Context) ([]*model.ClusterList, error)
	Close()

	Lockup(ctx context.Context, id uint64, prev *time.Time) error
	Unlock(ctx context.Context, id uint64) error
}

type clusterBaseRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) ClusterRepo {
	return &clusterBaseRepo{
		db: db,
	}
}

func (repo *clusterBaseRepo) Update(
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

func (repo *clusterBaseRepo) Delete(ctx context.Context, clusterId uint64) error {
	result := repo.db.Unscoped().Delete(&model.ClusterModel{}, clusterId)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

func (repo *clusterBaseRepo) DeleteByCreator(ctx context.Context, userId uint64) error {
	result := repo.db.Exec("delete from clusters where user_id = ? and deleted_at is null", userId)
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
	repo.db.Raw(
		"select c.id,c.kubeconfig,c.name,c.server,c.storage_class,c.info,c.user_id,c.created_at,c.inspect_at,count" +
			"(distinct cu.id) as users_count from clusters as c left join clusters_users as cu on c.id=cu.cluster_id" +
			" where c.deleted_at is null and cu.deleted_at is null group by c.id",
	).
		Scan(&result)
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

func (repo *clusterBaseRepo) Lockup(
	_ context.Context, id uint64, prev *time.Time,
) error {
	cm := model.ClusterModel{}
	db := repo.db.Model(&cm)
	if prev == nil {
		db = db.
			Where("`id` = ? and `inspect_at` is NULL", id).
			Update("inspect_at", time.Now().UTC())
	} else {
		db = db.
			Where("`id` = ? and `inspect_at` = ?", id, prev).
			Update("inspect_at", time.Now().UTC())
	}
	if db.Error != nil {
		return db.Error
	}
	if db.RowsAffected > 0 {
		return nil
	}
	return errors.New("0 rows affected")
}
func (repo *clusterBaseRepo) Unlock(_ context.Context, id uint64) error {
	cm := model.ClusterModel{}
	// Mariadb: if a NULL value is assigned to a TIMESTAMP field, the current date and time is assigned instead.
	// https://mariadb.com/kb/en/null-values/
	db := repo.db.Model(&cm).
		Where("`id` = ?", id).
		Update("inspect_at", time.Date(2014, 6, 7, 0, 0, 0, 0, time.UTC))
	return db.Error
}

// Close close db
func (repo *clusterBaseRepo) Close() {
	repo.db.Close()
}
