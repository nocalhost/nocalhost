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
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_user"
)

type ApplicationUserService interface {
	ListByApplicationId(ctx context.Context, applicationId uint64) ([]*model.ApplicationUserModel, error)
	ListByUserId(ctx context.Context, userId uint64) ([]*model.ApplicationUserModel, error)
	BatchDelete(ctx context.Context, applicationId uint64, userIds []uint64) error
	BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error
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

func (srv *applicationUserService) ListByApplicationId(ctx context.Context, id uint64) ([]*model.ApplicationUserModel, error) {
	return srv.repo.ListByApplicationId(ctx, id)
}

func (srv *applicationUserService) ListByUserId(ctx context.Context, id uint64) ([]*model.ApplicationUserModel, error) {
	return srv.repo.ListByUserId(ctx, id)
}

func (srv *applicationUserService) Close() {
	srv.repo.Close()
}
