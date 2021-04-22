/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package application_user

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_user"
)

type ApplicationUserService interface {
	ListByApplicationId(ctx context.Context, applicationId uint64) ([]*model.ApplicationUserModel, error)
	ListByUserId(ctx context.Context, userId uint64) ([]*model.ApplicationUserModel, error)
	BatchDelete(ctx context.Context, applicationId uint64, userIds []uint64) error
	BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error
	GetByApplicationIdAndUserId(ctx context.Context, applicationId uint64, userId uint64) (
		*model.ApplicationUserModel, error,
	)
	Close()
}

type applicationUserService struct {
	repo application_user.ApplicationUserRepo
}

func NewApplicationUserService() ApplicationUserService {
	db := model.GetDB()
	return &applicationUserService{repo: application_user.NewApplicationUserRepo(db)}
}

func (srv *applicationUserService) BatchDelete(ctx context.Context, applicationId uint64, userIds []uint64) error {
	return srv.repo.BatchDelete(ctx, applicationId, userIds)
}

func (srv *applicationUserService) BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error {
	return srv.repo.BatchInsert(ctx, applicationId, userIds)
}

func (srv *applicationUserService) GetByApplicationIdAndUserId(
	ctx context.Context, applicationId uint64, userId uint64,
) (*model.ApplicationUserModel, error) {
	return srv.repo.GetByApplicationIdAndUserId(ctx, applicationId, userId)
}

func (srv *applicationUserService) ListByApplicationId(ctx context.Context, id uint64) (
	[]*model.ApplicationUserModel, error,
) {
	return srv.repo.ListByApplicationId(ctx, id)
}

func (srv *applicationUserService) ListByUserId(ctx context.Context, id uint64) ([]*model.ApplicationUserModel, error) {
	return srv.repo.ListByUserId(ctx, id)
}

func (srv *applicationUserService) Close() {
	srv.repo.Close()
}
