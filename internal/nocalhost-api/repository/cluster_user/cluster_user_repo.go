/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
)

type ClusterUserRepo interface {
	Create(model model.ClusterUserModel) (model.ClusterUserModel, error)
	Delete(id uint64) error
	Modify(id uint64, attrs map[string]interface{}) error
	DeleteByWhere(models model.ClusterUserModel) error
	BatchDelete(id []uint64) error
	GetFirst(models model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetJoinCluster(condition model.ClusterUserJoinCluster) ([]*model.ClusterUserJoinCluster, error)
	GetList(models model.ClusterUserModel) ([]*model.ClusterUserModel, error)
	ListWithFuzzySpaceName(models model.ClusterUserModel) ([]*model.ClusterUserModel, error)
	Update(models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	UpdateKubeConfig(models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetJoinClusterAndAppAndUser(
		condition model.ClusterUserJoinClusterAndAppAndUser,
	) ([]*model.ClusterUserJoinClusterAndAppAndUser, error)
	GetJoinClusterAndAppAndUserDetail(
		condition model.ClusterUserJoinClusterAndAppAndUser,
	) (*model.ClusterUserJoinClusterAndAppAndUser, error)
	ListByUser(userId uint64) ([]*model.ClusterUserPluginModel, error)
	Close()
}

type clusterUserRepo struct {
	db *gorm.DB
}

func NewApplicationClusterRepo(db *gorm.DB) ClusterUserRepo {
	return &clusterUserRepo{
		db: db,
	}
}

// UpdateKubeConfig Deprecated
func (repo *clusterUserRepo) UpdateKubeConfig(
	models *model.ClusterUserModel,
) (*model.ClusterUserModel, error) {
	affect := repo.db.Model(&model.ClusterUserModel{}).Where("id=?", models.ID).Update(
		"kubeconfig", models.KubeConfig,
	).RowsAffected
	if affect > 0 {
		return models, nil
	}
	return models, errors.New("update fail")
}

// GetJoinCluster Get cluster user join users
func (repo *clusterUserRepo) GetJoinCluster(
	condition model.ClusterUserJoinCluster,
) ([]*model.ClusterUserJoinCluster, error) {
	var cluserUserJoinCluster []*model.ClusterUserJoinCluster
	result := repo.db.Table("clusters_users as cluster_user_join_clusters").Select(
		"cluster_user_join_clusters.id,cluster_user_join_clusters.application_id," +
			"cluster_user_join_clusters.user_id,cluster_user_join_clusters.cluster_id," +
			"cluster_user_join_clusters.namespace,c.name as admin_cluster_name,c.kubeconfig" +
			" as admin_cluster_kubeconfig",
	).
		Joins("join clusters as c on cluster_user_join_clusters.cluster_id=c.id").
		Where(condition).
		Scan(&cluserUserJoinCluster)
	if result.Error != nil {
		return cluserUserJoinCluster, result.Error
	}
	return cluserUserJoinCluster, nil
}

// DeleteByWhere
func (repo *clusterUserRepo) DeleteByWhere(models model.ClusterUserModel) error {
	result := repo.db.Unscoped().Delete(models)
	return result.Error
}

// BatchDelete
func (repo *clusterUserRepo) BatchDelete(ids []uint64) error {
	result := repo.db.Unscoped().Delete(model.ClusterUserModel{}, ids)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

// Delete
func (repo *clusterUserRepo) Delete(id uint64) error {
	result := repo.db.Unscoped().Delete(model.ClusterUserModel{}, id)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

// Update
func (repo *clusterUserRepo) Update(models *model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	where := model.ClusterUserModel{
		ApplicationId: models.ApplicationId,
		UserId:        models.UserId,
	}
	_, err := repo.GetFirst(where)
	if err != nil {
		return models, errors.Wrap(err, "[clsuter_user_repo] get clsuter_user denied")
	}
	emptyModel := model.ClusterUserModel{}
	affectRow := repo.db.Model(&emptyModel).Where("id=?", models.ID).Update(models).RowsAffected
	if affectRow > 0 {
		return models, nil
	}
	return models, errors.New("update dev space err")
}

func (repo *clusterUserRepo) Modify(id uint64, attrs map[string]interface{}) error {
	criteria := model.ClusterUserModel{ID: id}
	affected := repo.db.Model(&criteria).Update(attrs).RowsAffected
	if affected == 0 {
		return errors.New("0 rows affected")
	}
	return nil
}

// GetList
func (repo *clusterUserRepo) GetList(models model.ClusterUserModel) (
	[]*model.ClusterUserModel, error,
) {
	result := make([]*model.ClusterUserModel, 0)
	repo.db.Where(&models).Order("cluster_admin desc, user_id asc").Find(&result)
	if len(result) > 0 {
		return result, nil
	}
	return nil, errors.New("users cluster not found")
}

// ListWithFuzzySpaceName
func (repo *clusterUserRepo) ListWithFuzzySpaceName(models model.ClusterUserModel) (
	[]*model.ClusterUserModel, error,
) {
	condition := repo.db.Where(&models)
	if models.SpaceName != "" {
		condition = condition.Where("space_name like ?", "%"+models.SpaceName+"%")
		models.SpaceName = ""
	}

	result := make([]*model.ClusterUserModel, 0)
	condition.Order("cluster_admin desc, user_id asc").Find(&result)
	if len(result) > 0 {
		return result, nil
	}
	return nil, errors.New("users cluster not found")
}

// ListByUser
func (repo *clusterUserRepo) ListByUser(userId uint64) ([]*model.ClusterUserPluginModel, error) {
	result := make([]*model.ClusterUserPluginModel, 0)

	repo.db.Raw("SELECT * FROM clusters_users WHERE user_id = ? ORDER BY user_id, id", userId).Scan(&result)
	if len(result) > 0 {
		return result, nil
	}
	return nil, errors.New("users cluster not found")
}

// GetFirst
func (repo *clusterUserRepo) GetFirst(models model.ClusterUserModel) (
	*model.ClusterUserModel, error,
) {
	cluster := model.ClusterUserModel{}
	result := repo.db.Where(&models).First(&cluster)
	if result.Error != nil {
		return &cluster, result.Error
	}
	return &cluster, nil
}

// Create
func (repo *clusterUserRepo) Create(model model.ClusterUserModel) (model.ClusterUserModel, error) {
	err := repo.db.Create(&model).Error
	if err != nil {
		return model, errors.Wrap(err, "[application_cluster_repo] create application_cluster error")
	}

	return model, nil
}

// GetJoinClusterAndAppAndUser
func (repo *clusterUserRepo) GetJoinClusterAndAppAndUser(
	condition model.ClusterUserJoinClusterAndAppAndUser,
) ([]*model.ClusterUserJoinClusterAndAppAndUser, error) {
	var result []*model.ClusterUserJoinClusterAndAppAndUser
	sqlResult := repo.db.Table("clusters_users as cluster_user_join_cluster_and_app_and_users").
		Select(
			"cluster_user_join_cluster_and_app_and_users.id," +
				"cluster_user_join_cluster_and_app_and_users.application_id," +
				"a.context as application_name, cluster_user_join_cluster_and_app_and_users.user_id," +
				" u.name as user_name, cluster_user_join_cluster_and_app_and_users.cluster_id, c.name as" +
				" cluster_name,cluster_user_join_cluster_and_app_and_users.namespace," +
				"cluster_user_join_cluster_and_app_and_users.space_name," +
				"cluster_user_join_cluster_and_app_and_users.kubeconfig," +
				"cluster_user_join_cluster_and_app_and_users.space_resource_limit," +
				"cluster_user_join_cluster_and_app_and_users.status," +
				"cluster_user_join_cluster_and_app_and_users.created_at",
		).Joins("left join users as u on cluster_user_join_cluster_and_app_and_users.user_id=u.id").
		Joins("left join applications as a on cluster_user_join_cluster_and_app_and_users.application_id=a.id").
		Joins("left join clusters as c on cluster_user_join_cluster_and_app_and_users.cluster_id=c.id").
		Where(condition).
		Scan(&result)
	if sqlResult.Error != nil {
		return result, sqlResult.Error
	}
	return result, nil
}

// GetJoinClusterAndAppAndUserDetail
func (repo *clusterUserRepo) GetJoinClusterAndAppAndUserDetail(
	condition model.ClusterUserJoinClusterAndAppAndUser,
) (*model.ClusterUserJoinClusterAndAppAndUser, error) {
	result := model.ClusterUserJoinClusterAndAppAndUser{}
	sqlResult := repo.db.Table("clusters_users as cluster_user_join_cluster_and_app_and_users").
		Select(
			"cluster_user_join_cluster_and_app_and_users.id," +
				"cluster_user_join_cluster_and_app_and_users.cluster_admin," +
				"cluster_user_join_cluster_and_app_and_users.application_id," +
				"cluster_user_join_cluster_and_app_and_users.user_id, u.name as user_name," +
				"cluster_user_join_cluster_and_app_and_users.cluster_id, " +
				"c.name as cluster_name,cluster_user_join_cluster_and_app_and_users.namespace," +
				"cluster_user_join_cluster_and_app_and_users.space_name," +
				"cluster_user_join_cluster_and_app_and_users.kubeconfig," +
				"cluster_user_join_cluster_and_app_and_users.space_resource_limit," +
				"cluster_user_join_cluster_and_app_and_users.status," +
				"cluster_user_join_cluster_and_app_and_users.created_at",
		).
		Joins(
			"left join users as u on " +
				"cluster_user_join_cluster_and_app_and_users.user_id=u.id",
		).Joins(
		"left join clusters as c on cluster_user_join_cluster_and_app_and_users." +
			"cluster_id=c.id",
	).Where(condition).Scan(&result)
	if sqlResult.Error != nil {
		return &result, sqlResult.Error
	}

	return &result, nil
}

// Close close db
func (repo *clusterUserRepo) Close() {
	repo.db.Close()
}
