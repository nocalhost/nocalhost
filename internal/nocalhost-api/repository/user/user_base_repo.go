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

// BaseRepo 定义接口
type BaseRepo interface {
	Create(ctx context.Context, user model.UserBaseModel) (id uint64, err error)
	Update(ctx context.Context, id uint64, userMap map[string]interface{}) error
	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
	Close()
}

// userBaseRepo 用户仓库
type userBaseRepo struct {
	db *gorm.DB
}

// NewUserRepo 实例化用户仓库
func NewUserRepo(db *gorm.DB) BaseRepo {
	return &userBaseRepo{
		db: db,
	}
}

// Create 创建用户
func (repo *userBaseRepo) Create(ctx context.Context, user model.UserBaseModel) (id uint64, err error) {
	err = repo.db.Create(&user).Error
	if err != nil {
		return 0, errors.Wrap(err, "[user_repo] create user err")
	}

	return user.ID, nil
}

// Update 更新用户信息
func (repo *userBaseRepo) Update(ctx context.Context, id uint64, userMap map[string]interface{}) error {
	user, err := repo.GetUserByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "[user_repo] update user data err")
	}
	return repo.db.Model(user).Updates(userMap).Error
}

// GetUserByID 获取用户
func (repo *userBaseRepo) GetUserByID(ctx context.Context, uid uint64) (userBase *model.UserBaseModel, err error) {
	start := time.Now()
	defer func() {
		log.Infof("[repo.user_base] get user by uid: %d cost: %d ns", uid, time.Now().Sub(start).Nanoseconds())
	}()

	data := new(model.UserBaseModel)

	// 从数据库中获取
	err = repo.db.First(data, uid).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(err, "[repo.user_base] get user data err")
	}
	return data, nil
}

// GetUserByPhone 根据手机号获取用户
func (repo *userBaseRepo) GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error) {
	user := model.UserBaseModel{}
	err := repo.db.Where("phone = ?", phone).First(&user).Error
	if err != nil {
		return nil, errors.Wrap(err, "[user_repo] get user err by phone")
	}

	return &user, nil
}

// GetUserByEmail 根据邮箱获取手机号
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
