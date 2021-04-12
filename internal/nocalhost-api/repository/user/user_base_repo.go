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
package user

import (
	"context"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// BaseRepo
type BaseRepo interface {
	Create(ctx context.Context, user model.UserBaseModel) (model.UserBaseModel, error)
	Update(ctx context.Context, id uint64, userMap *model.UserBaseModel) (*model.UserBaseModel, error)
	Delete(ctx context.Context, id uint64) error
	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
	GetUserList(ctx context.Context) ([]*model.UserList, error)
	UpdateServiceAccountName(ctx context.Context, id uint64, saName string) error
	Close()
}

// userBaseRepo
type userBaseRepo struct {
	db *gorm.DB
}

// NewUserRepo
func NewUserRepo(db *gorm.DB) BaseRepo {
	return &userBaseRepo{
		db: db,
	}
}

// GetUserList
func (repo *userBaseRepo) GetUserList(ctx context.Context) ([]*model.UserList, error) {
	var result []*model.UserList
	repo.db.Raw("select u.id as id,u.name as name,u.email as email,count(distinct cu.id) as cluster_count,u.status as status, u.is_admin as is_admin from users as u left join clusters_users as cu on cu.user_id=u.id where u.deleted_at is null and cu.deleted_at is null group by u.id").Scan(&result)
	return result, nil
}

// Delete
func (repo *userBaseRepo) Delete(ctx context.Context, id uint64) error {
	users := model.UserBaseModel{
		ID: id,
	}
	if result := repo.db.Where("id=?", id).Unscoped().Delete(&users); result.RowsAffected > 0 {
		return nil
	}
	return errors.New("user delete fail")
}

// Create
func (repo *userBaseRepo) Create(ctx context.Context, user model.UserBaseModel) (model.UserBaseModel, error) {
	err := repo.db.Create(&user).Error
	if err != nil {
		return user, errors.Wrap(err, "[user_repo] create user err")
	}

	return user, nil
}

// Update
func (repo *userBaseRepo) Update(ctx context.Context, id uint64, userMap *model.UserBaseModel) (*model.UserBaseModel, error) {
	user, err := repo.GetUserByID(ctx, id)
	if user.ID != id {
		return user, errors.New("[user_repo] user is not exsit.")
	}
	if err != nil {
		return user, errors.Wrap(err, "[user_repo] update user data err")
	}
	err = repo.db.Model(&user).Updates(&userMap).Where("id=?", id).Error
	if err != nil {
		return user, errors.Wrap(err, "[user_repo] update user data error")
	}
	return user, nil
}

// Update
func (repo *userBaseRepo) UpdateServiceAccountName(ctx context.Context, id uint64, saName string) error {
	if err := repo.db.Exec("UPDATE users SET sa_name = ? WHERE id = ?", saName, id).Error; err != nil {
		return err
	}

	return nil
}

// GetUserByID
func (repo *userBaseRepo) GetUserByID(ctx context.Context, uid uint64) (userBase *model.UserBaseModel, err error) {
	start := time.Now()
	defer func() {
		log.Infof("[repo.user_base] get user by uid: %d cost: %d ns", uid, time.Now().Sub(start).Nanoseconds())
	}()

	data := new(model.UserBaseModel)

	err = repo.db.First(data, uid).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, "[repo.user_base] get user data err")
	}
	return data, nil
}

// GetUserByPhone
func (repo *userBaseRepo) GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error) {
	user := model.UserBaseModel{}
	err := repo.db.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, errors.Wrap(err, "[user_repo] get user err by phone")
	}

	return &user, nil
}

// GetUserByEmail
func (repo *userBaseRepo) GetUserByEmail(ctx context.Context, phone string) (*model.UserBaseModel, error) {
	user := model.UserBaseModel{}
	err := repo.db.Where("email = ?", phone).First(&user).Error
	if err != nil {
		return nil, errors.Wrap(err, "[user_repo] get user err by email")
	}

	return &user, nil
}

// Close close db
func (repo *userBaseRepo) Close() {
	repo.db.Close()
}
