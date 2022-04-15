/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package application

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type ApplicationRepo struct {
	db *gorm.DB
}

func NewClusterRepo(db *gorm.DB) *ApplicationRepo {
	return &ApplicationRepo{
		db: db,
	}
}

func (repo *ApplicationRepo) PublicSwitch(ctx context.Context, applicationId uint64, public uint8) error {
	if err := repo.db.Exec(
		"UPDATE applications SET public = ? "+
			"WHERE id = ?", public, applicationId,
	).Error; err != nil {
		return err
	}

	return nil
}

func (repo *ApplicationRepo) GetByName(ctx context.Context, name string) (model.ApplicationModel, error) {
	var record model.ApplicationModel
	result := repo.db.Where("JSON_CONTAINS(context,JSON_OBJECT('application_name', ?))", name).
		First(&record)
	if result.Error != nil {
		return record, nil
	}
	return record, nil
}

func (repo *ApplicationRepo) PluginGetList(ctx context.Context, userId uint64) (
	[]*model.PluginApplicationModel, error,
) {
	var result []*model.PluginApplicationModel
	repo.db.Table("applications").
		Select(
			"clusters.storage_class,applications.id,applications.context,applications.user_id,"+
				"applications.status,clusters_users.cluster_id,clusters_users.space_name,clusters_users.kubeconfig,"+
				"clusters_users.memory,clusters_users.cpu,clusters_users.namespace,clusters_users.status as"+
				" install_status,clusters_users.id as devspace_id",
		).Joins(
		"join clusters_users on applications.id=clusters_users.application_id and clusters_users.user_id=?"+
			" join clusters on clusters.id=clusters_users.cluster_id",
		userId,
	).Scan(&result)
	return result, nil
}

func (repo *ApplicationRepo) Create(ctx context.Context, application model.ApplicationModel) (
	model.ApplicationModel, error,
) {
	err := repo.db.Create(&application).Error
	if err != nil {
		return application, errors.Wrap(err, "[application_repo] create application err")
	}

	return application, nil
}

func (repo *ApplicationRepo) Get(ctx context.Context, id uint64) (model.ApplicationModel, error) {
	// Here is the Struct type, and Error will be thrown when the data is not available
	// If the input is of the make([]*model.ApplicationModel,0)
	// Slice type, then Error will never be thrown if no data is available
	application := model.ApplicationModel{}
	result := repo.db.Where("status=1 and id=?", id).First(&application)
	if err := result.Error; err != nil {
		log.Warnf("[application_repo] get application id: %v error", id)
		return application, err
	}
	return application, nil
}

func (repo *ApplicationRepo) GetList(ctx context.Context, userId *uint64) ([]*model.ApplicationModel, error) {
	applicationList := make([]*model.ApplicationModel, 0)

	query := repo.db.Where("status = 1")

	if userId != nil {
		query.Where("user_id = ?", &userId)
	}

	result := query.Find(&applicationList)

	if err := result.Error; err != nil {
		log.Warnf("[application_repo] get application err")
		return nil, err
	}

	return applicationList, nil
}

func (repo *ApplicationRepo) Delete(ctx context.Context, id uint64) error {
	application := model.ApplicationModel{
		ID: id,
	}
	if result := repo.db.Unscoped().Delete(&application); result.RowsAffected > 0 {
		return nil
	}
	return errors.New("application delete denied")
}

func (repo *ApplicationRepo) Update(
	ctx context.Context, applicationModel *model.ApplicationModel,
) (*model.ApplicationModel, error) {
	application, err := repo.Get(ctx, applicationModel.ID)
	if application.ID != applicationModel.ID {
		return applicationModel, errors.New("[application_repo] application is not exsit.")
	}
	if err != nil {
		return applicationModel, errors.Wrap(err, "[application_repo] get application denied")
	}
	affectRow := repo.
		db.
		Model(&application).
		Update(&applicationModel).
		Where("id=?", application.ID).
		RowsAffected

	if affectRow > 0 {
		return applicationModel, nil
	}
	return applicationModel, errors.Wrap(err, "[application_repo] update application err")
}

// Close close db
func (repo *ApplicationRepo) Close() {
	repo.db.Close()
}
