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

package application_user

import (
	"context"
	"errors"
	"nocalhost/internal/nocalhost-api/model"

	"github.com/jinzhu/gorm"
)

type ApplicationUserRepo interface {
	ListByApplicationId(ctx context.Context, applicationId uint64) ([]*model.ApplicationUserModel, error)
	ListByUserId(ctx context.Context, userId uint64) ([]*model.ApplicationUserModel, error)
	BatchDelete(ctx context.Context, applicationId uint64, userIds []uint64) error
	BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error
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

func (repo *applicationUserRepo) BatchDelete(ctx context.Context, applicationId uint64, userIds []uint64) error {
	if len(userIds) == 0 {
		return errors.New("Can not batch delete applications_users with empty userIds ")
	}

	if err := repo.db.Exec("DELETE FROM applications_users WHERE application_id = ? AND user_id IN (?)", applicationId, userIds).Error; err != nil {
		return err
	}
	return nil
}

func (repo *applicationUserRepo) BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error {
	if len(userIds) == 0 {
		return errors.New("Can not batch insert applications_users with empty userIds ")
	}

	var result []model.ApplicationUserModel

	for _, userId := range userIds {
		result = append(result,
			model.ApplicationUserModel{
				ApplicationId: applicationId,
				UserId:        userId,
			},
		)
	}

	if err := repo.db.Create(result).Error; err != nil {
		return err
	}
	return nil
}

func (repo *applicationUserRepo) ListByApplicationId(ctx context.Context, applicationId uint64) ([]*model.ApplicationUserModel, error) {
	var result []*model.ApplicationUserModel
	err := repo.db.Where("application_id=?", applicationId).Find(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

func (repo *applicationUserRepo) ListByUserId(ctx context.Context, userId uint64) ([]*model.ApplicationUserModel, error) {
	var result []*model.ApplicationUserModel
	err := repo.db.Where("user_id=?", userId).Find(&result)
	if err.Error != nil {
		return result, err.Error
	}
	return result, nil
}

// Close close db
func (repo *applicationUserRepo) Close() {
	repo.db.Close()
}
