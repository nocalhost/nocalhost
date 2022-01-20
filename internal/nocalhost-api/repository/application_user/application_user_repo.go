/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package application_user

import (
	"errors"
	"github.com/jinzhu/gorm"
	"nocalhost/internal/nocalhost-api/model"
)

type ApplicationUserRepo interface {
	GetByApplicationIdAndUserId(applicationId uint64, userId uint64) (
		*model.ApplicationUserModel, error,
	)
	ListByApplicationIdFromRepo(applicationId uint64) ([]*model.ApplicationUserModel, error)
	ListByUserIdFromRepo(userId uint64) ([]*model.ApplicationUserModel, error)
	BatchDeleteFromRepo(applicationId uint64, userIds []uint64) error
	BatchInsertIntoRepo(applicationId uint64, userIds []uint64) error
	Close()
}

type applicationUserRepo struct {
	db *gorm.DB
}

func NewApplicationUserRepo(db *gorm.DB) ApplicationUserRepo {
	return &applicationUserRepo{
		db: db,
	}
}

func (repo *applicationUserRepo) BatchDeleteFromRepo(applicationId uint64, userIds []uint64) error {
	if len(userIds) == 0 {
		return errors.New("Can not batch delete applications_users with empty userIds ")
	}

	if err := repo.db.Exec(
		"DELETE FROM applications_users WHERE application_id = ? AND user_id IN (?)", applicationId, userIds,
	).Error; err != nil {
		return err
	}
	return nil
}

func (repo *applicationUserRepo) BatchInsertIntoRepo(applicationId uint64, userIds []uint64) error {
	if len(userIds) == 0 {
		return errors.New("Can not batch insert applications_users with empty userIds ")
	}

	for _, userId := range userIds {
		repo.db.Create(
			&model.ApplicationUserModel{
				ApplicationId: applicationId,
				UserId:        userId,
			},
		)
	}

	return nil
}

func (repo *applicationUserRepo) ListByApplicationIdFromRepo(
	applicationId uint64,
) ([]*model.ApplicationUserModel, error) {
	var result []*model.ApplicationUserModel
	err := repo.db.Where("application_id=?", applicationId).Find(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *applicationUserRepo) ListByUserIdFromRepo(userId uint64) (
	[]*model.ApplicationUserModel, error,
) {
	var result []*model.ApplicationUserModel
	err := repo.db.Where("user_id=?", userId).Find(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *applicationUserRepo) GetByApplicationIdAndUserId(
	applicationId uint64, userId uint64,
) (*model.ApplicationUserModel, error) {
	var result = model.ApplicationUserModel{}
	err := repo.db.Where(
		&model.ApplicationUserModel{
			ApplicationId: applicationId,
			UserId:        userId,
		},
	).First(&result).Error

	return &result, err
}

// Close close db
func (repo *applicationUserRepo) Close() {
	repo.db.Close()
}
