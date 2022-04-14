/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package application_user

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_user"
)

type ApplicationUser struct {
	repo *application_user.ApplicationUserRepoBase
}

func NewApplicationUserService() *ApplicationUser {
	db := model.GetDB()
	return &ApplicationUser{repo: application_user.NewApplicationUserRepo(db)}
}

func (srv *ApplicationUser) BatchDelete(ctx context.Context, applicationId uint64, userIds []uint64) error {
	return srv.repo.BatchDeleteFromRepo(applicationId, userIds)
}

func (srv *ApplicationUser) BatchInsert(ctx context.Context, applicationId uint64, userIds []uint64) error {
	return srv.repo.BatchInsertIntoRepo(applicationId, userIds)
}

func (srv *ApplicationUser) GetByApplicationIdAndUserId(
	ctx context.Context, applicationId uint64, userId uint64,
) (*model.ApplicationUserModel, error) {
	return srv.repo.GetByApplicationIdAndUserId(applicationId, userId)
}

func (srv *ApplicationUser) ListByApplicationId(ctx context.Context, id uint64) (
	[]*model.ApplicationUserModel, error,
) {
	return srv.repo.ListByApplicationIdFromRepo(id)
}

func (srv *ApplicationUser) ListByUserId(ctx context.Context, id uint64) ([]*model.ApplicationUserModel, error) {
	return srv.repo.ListByUserIdFromRepo(id)
}

func (srv *ApplicationUser) Close() {
	srv.repo.Close()
}
