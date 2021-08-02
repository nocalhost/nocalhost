/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
	return srv.repo.BatchDeleteFromRepo(applicationId, userIds)
}

func (srv *applicationUserService) BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error {
	return srv.repo.BatchInsertIntoRepo(applicationId, userIds)
}

func (srv *applicationUserService) GetByApplicationIdAndUserId(
	ctx context.Context, applicationId uint64, userId uint64,
) (*model.ApplicationUserModel, error) {
	return srv.repo.GetByApplicationIdAndUserId(applicationId, userId)
}

func (srv *applicationUserService) ListByApplicationId(ctx context.Context, id uint64) (
	[]*model.ApplicationUserModel, error,
) {
	return srv.repo.ListByApplicationIdFromRepo(id)
}

func (srv *applicationUserService) ListByUserId(ctx context.Context, id uint64) ([]*model.ApplicationUserModel, error) {
	return srv.repo.ListByUserIdFromRepo(id)
}

func (srv *applicationUserService) Close() {
	srv.repo.Close()
}
