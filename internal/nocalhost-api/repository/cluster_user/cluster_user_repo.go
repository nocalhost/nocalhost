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

package cluster_user

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/model"
)

type ClusterUserRepo interface {
	Create(ctx context.Context, model model.ClusterUserModel) (model.ClusterUserModel, error)
	Delete(ctx context.Context, id uint64) error
	DeleteByWhere(ctx context.Context, models model.ClusterUserModel) error
	BatchDelete(ctx context.Context, id []uint64) error
	GetFirst(ctx context.Context, models model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetJoinCluster(ctx context.Context, condition model.ClusterUserJoinCluster) ([]*model.ClusterUserJoinCluster, error)
	GetList(ctx context.Context, models model.ClusterUserModel) ([]*model.ClusterUserModel, error)
	Update(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	UpdateKubeConfig(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error)
	GetJoinClusterAndAppAndUser(ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser) ([]*model.ClusterUserJoinClusterAndAppAndUser, error)
	GetJoinClusterAndAppAndUserDetail(ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser) (*model.ClusterUserJoinClusterAndAppAndUser, error)
	ListByUser(ctx context.Context, userId uint64) ([]*model.ClusterUserPluginModel, error)
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

func (repo *clusterUserRepo) UpdateKubeConfig(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error) {
	affect := repo.db.Model(&model.ClusterUserModel{}).Where("id=?", models.ID).Update("kubeconfig", models.KubeConfig).RowsAffected
	if affect > 0 {
		return models, nil
	}
	return models, errors.New("update fail")
}

func (repo *clusterUserRepo) GetJoinCluster(ctx context.Context, condition model.ClusterUserJoinCluster) ([]*model.ClusterUserJoinCluster, error) {
	var cluserUserJoinCluster []*model.ClusterUserJoinCluster
	result := repo.db.Table("clusters_users as cluster_user_join_clusters").Select("cluster_user_join_clusters.id,cluster_user_join_clusters.application_id,cluster_user_join_clusters.user_id,cluster_user_join_clusters.cluster_id,cluster_user_join_clusters.namespace,c.name as admin_cluster_name,c.kubeconfig as admin_cluster_kubeconfig").Joins("join clusters as c on cluster_user_join_clusters.cluster_id=c.id").Where(condition).Scan(&cluserUserJoinCluster)
	if result.Error != nil {
		return cluserUserJoinCluster, result.Error
	}
	return cluserUserJoinCluster, nil
}

func (repo *clusterUserRepo) DeleteByWhere(ctx context.Context, models model.ClusterUserModel) error {
	result := repo.db.Unscoped().Delete(models)
	return result.Error
}

func (repo *clusterUserRepo) BatchDelete(ctx context.Context, ids []uint64) error {
	result := repo.db.Unscoped().Delete(model.ClusterUserModel{}, ids)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

func (repo *clusterUserRepo) Delete(ctx context.Context, id uint64) error {
	result := repo.db.Unscoped().Delete(model.ClusterUserModel{}, id)
	if result.RowsAffected > 0 {
		return nil
	}
	return result.Error
}

func (repo *clusterUserRepo) Update(ctx context.Context, models *model.ClusterUserModel) (*model.ClusterUserModel, error) {
	where := model.ClusterUserModel{
		ApplicationId: models.ApplicationId,
		UserId:        models.UserId,
	}
	_, err := repo.GetFirst(ctx, where)
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

func (repo *clusterUserRepo) GetList(ctx context.Context, models model.ClusterUserModel) ([]*model.ClusterUserModel, error) {
	result := make([]*model.ClusterUserModel, 0)
	repo.db.Where(&models).Find(&result)
	if len(result) > 0 {
		return result, nil
	}
	return nil, errors.New("users cluster not found")
}

func (repo *clusterUserRepo) ListByUser(ctx context.Context, userId uint64) ([]*model.ClusterUserPluginModel, error) {
	result := make([]*model.ClusterUserPluginModel, 0)

	repo.db.Raw("SELECT * FROM clusters_users WHERE user_id = ? ORDER BY user_id, id", userId).Scan(&result)
	if len(result) > 0 {
		return result, nil
	}
	return nil, errors.New("users cluster not found")
}

func (repo *clusterUserRepo) GetFirst(ctx context.Context, models model.ClusterUserModel) (*model.ClusterUserModel, error) {
	cluster := model.ClusterUserModel{}
	result := repo.db.Where(&models).First(&cluster)
	if result.Error != nil {
		return &cluster, result.Error
	}
	return &cluster, nil
}

func (repo *clusterUserRepo) Create(ctx context.Context, model model.ClusterUserModel) (model.ClusterUserModel, error) {
	err := repo.db.Create(&model).Error
	if err != nil {
		return model, errors.Wrap(err, "[application_cluster_repo] create application_cluster error")
	}

	return model, nil
}

func (repo *clusterUserRepo) GetJoinClusterAndAppAndUser(ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser) ([]*model.ClusterUserJoinClusterAndAppAndUser, error) {
	var result []*model.ClusterUserJoinClusterAndAppAndUser
	sqlResult := repo.db.Table("clusters_users as cluster_user_join_cluster_and_app_and_users").Select("cluster_user_join_cluster_and_app_and_users.id ,cluster_user_join_cluster_and_app_and_users.application_id,a.context as application_name, cluster_user_join_cluster_and_app_and_users.user_id, u.name as user_name, cluster_user_join_cluster_and_app_and_users.cluster_id, c.name as cluster_name,cluster_user_join_cluster_and_app_and_users.namespace,cluster_user_join_cluster_and_app_and_users.space_name,cluster_user_join_cluster_and_app_and_users.kubeconfig,cluster_user_join_cluster_and_app_and_users.space_resource_limit,cluster_user_join_cluster_and_app_and_users.status,cluster_user_join_cluster_and_app_and_users.created_at").Joins("left join users as u on cluster_user_join_cluster_and_app_and_users.user_id=u.id").Joins("left join applications as a on cluster_user_join_cluster_and_app_and_users.application_id=a.id").Joins("left join clusters as c on cluster_user_join_cluster_and_app_and_users.cluster_id=c.id").Where(condition).Scan(&result)
	if sqlResult.Error != nil {
		return result, sqlResult.Error
	}
	return result, nil
}

func (repo *clusterUserRepo) GetJoinClusterAndAppAndUserDetail(ctx context.Context, condition model.ClusterUserJoinClusterAndAppAndUser) (*model.ClusterUserJoinClusterAndAppAndUser, error) {
	result := model.ClusterUserJoinClusterAndAppAndUser{}
	sqlResult := repo.db.Table("clusters_users as cluster_user_join_cluster_and_app_and_users").
		Select("cluster_user_join_cluster_and_app_and_users.id ,cluster_user_join_cluster_and_app_and_users.application_id, cluster_user_join_cluster_and_app_and_users.user_id, u.name as user_name, cluster_user_join_cluster_and_app_and_users.cluster_id, c.name as cluster_name,cluster_user_join_cluster_and_app_and_users.namespace,cluster_user_join_cluster_and_app_and_users.space_name,cluster_user_join_cluster_and_app_and_users.kubeconfig,cluster_user_join_cluster_and_app_and_users.space_resource_limit,cluster_user_join_cluster_and_app_and_users.status,cluster_user_join_cluster_and_app_and_users.created_at").Joins("left join users as u on cluster_user_join_cluster_and_app_and_users.user_id=u.id").Joins("left join clusters as c on cluster_user_join_cluster_and_app_and_users.cluster_id=c.id").Where(condition).Scan(&result)
	if sqlResult.Error != nil {
		return &result, sqlResult.Error
	}

	return &result, nil
}

// Close close db
func (repo *clusterUserRepo) Close() {
	repo.db.Close()
}
