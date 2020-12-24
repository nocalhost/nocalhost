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

package application

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ApplicationRepo interface {
	Create(ctx context.Context, application model.ApplicationModel) (model.ApplicationModel, error)
	Get(ctx context.Context, userId uint64, id uint64) (model.ApplicationModel, error)
	GetByName(ctx context.Context, name string) (model.ApplicationModel, error)
	GetList(ctx context.Context, userId uint64) ([]*model.ApplicationModel, error)
	PluginGetList(ctx context.Context, userId uint64) ([]*model.PluginApplicationModel, error)
	Delete(ctx context.Context, userId uint64, id uint64) error
	Update(ctx context.Context, applicationModel *model.ApplicationModel) (*model.ApplicationModel, error)
	Close()
}

type applicationRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) ApplicationRepo {
	return &applicationRepo{
		db: db,
	}
}

func (repo *applicationRepo) GetByName(ctx context.Context, name string) (model.ApplicationModel, error) {
	var record model.ApplicationModel
	result := repo.db.Where("JSON_CONTAINS(context,JSON_OBJECT('application_name', ?))", name).First(&record)
	if result.Error != nil {
		return record, nil
	}
	return record, nil
}

func (repo *applicationRepo) PluginGetList(ctx context.Context, userId uint64) ([]*model.PluginApplicationModel, error) {
	var result []*model.PluginApplicationModel
	repo.db.Table("applications").Select("clusters.storage_class,applications.id,applications.context,applications.user_id,applications.status,clusters_users.cluster_id,clusters_users.kubeconfig,clusters_users.memory,clusters_users.cpu,clusters_users.namespace,clusters_users.status as install_status,clusters_users.id as devspace_id").Joins("join clusters_users on applications.id=clusters_users.application_id and clusters_users.user_id=? join clusters on clusters.id=clusters_users.cluster_id", userId).Scan(&result)
	return result, nil
}

func (repo *applicationRepo) Create(ctx context.Context, application model.ApplicationModel) (model.ApplicationModel, error) {
	err := repo.db.Create(&application).Error
	if err != nil {
		return application, errors.Wrap(err, "[application_repo] create application err")
	}

	return application, nil
}

func (repo *applicationRepo) Get(ctx context.Context, userId uint64, id uint64) (model.ApplicationModel, error) {
	// Here is the Struct type, and Error will be thrown when the data is not available
	//If the input is of the make([]*model.ApplicationModel,0) Slice type, then Error will never be thrown if no data is available
	application := model.ApplicationModel{}
	result := repo.db.Where("user_id=? and status=1 and id=?", userId, id).First(&application)
	if err := result.Error; err != nil {
		log.Warnf("[application_repo] get application for user: %v id: %v error", userId, id)
		return application, err
	}
	return application, nil
}

func (repo *applicationRepo) GetList(ctx context.Context, userId uint64) ([]*model.ApplicationModel, error) {
	applicationList := make([]*model.ApplicationModel, 0)
	result := repo.db.Where("user_id=? and status=1", userId).Find(&applicationList)

	if err := result.Error; err != nil {
		log.Warnf("[application_repo] get application for user %s err", userId)
		return nil, err
	}

	return applicationList, nil
}

func (repo *applicationRepo) Delete(ctx context.Context, userId uint64, id uint64) error {
	application := model.ApplicationModel{
		ID: id,
	}
	if result := repo.db.Unscoped().Where("user_id=?", userId).Delete(&application); result.RowsAffected > 0 {
		return nil
	}
	return errors.New("application delete denied")
}

func (repo *applicationRepo) Update(ctx context.Context, applicationModel *model.ApplicationModel) (*model.ApplicationModel, error) {
	_, err := repo.Get(ctx, applicationModel.UserId, applicationModel.ID)
	if err != nil {
		return applicationModel, errors.Wrap(err, "[application_repo] get application denied")
	}
	affectRow := repo.db.Save(&applicationModel).RowsAffected
	if affectRow > 0 {
		return applicationModel, nil
	}
	return applicationModel, errors.Wrap(err, "[application_repo] update application err")
}

// Close close db
func (repo *applicationRepo) Close() {
	repo.db.Close()
}
