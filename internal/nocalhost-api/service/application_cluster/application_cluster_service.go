/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package application_cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_cluster"

	"github.com/pkg/errors"
)

type ApplicationClusterService interface {
	Create(ctx context.Context, applicationId uint64, clusterId uint64) (model.ApplicationClusterModel, error)
	GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error)
	GetList(ctx context.Context, id uint64) ([]*model.ApplicationClusterModel, error)
	GetJoinCluster(ctx context.Context, id uint64) ([]*model.ApplicationClusterJoinModel, error)
	Close()
}

type applicationClusterService struct {
	applicationClusterRepo application_cluster.ApplicationClusterRepo
}

func NewApplicationClusterService() ApplicationClusterService {
	db := model.GetDB()
	return &applicationClusterService{applicationClusterRepo: application_cluster.NewApplicationClusterRepo(db)}
}

func (srv *applicationClusterService) GetJoinCluster(
	ctx context.Context, id uint64,
) ([]*model.ApplicationClusterJoinModel, error) {
	return srv.applicationClusterRepo.GetJoinCluster(ctx, id)
}

func (srv *applicationClusterService) GetList(ctx context.Context, id uint64) (
	[]*model.ApplicationClusterModel, error,
) {
	return srv.applicationClusterRepo.GetList(ctx, id)
}

func (srv *applicationClusterService) GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error) {
	return srv.applicationClusterRepo.GetFirst(ctx, id)
}

func (srv *applicationClusterService) Create(
	ctx context.Context, applicationId uint64, clusterId uint64,
) (model.ApplicationClusterModel, error) {
	c := model.ApplicationClusterModel{
		ApplicationId: applicationId,
		ClusterId:     clusterId,
	}
	_, err := srv.applicationClusterRepo.Create(ctx, c)
	if err != nil {
		return c, errors.Wrapf(err, "create application_cluster")
	}
	return c, nil
}

func (srv *applicationClusterService) Close() {
	srv.applicationClusterRepo.Close()
}
