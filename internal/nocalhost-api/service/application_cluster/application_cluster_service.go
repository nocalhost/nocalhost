/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package application_cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/application_cluster"

	"github.com/pkg/errors"
)

type ApplicationCluster struct {
	applicationClusterRepo *application_cluster.ApplicationClusterRepoBase
}

func NewApplicationClusterService() *ApplicationCluster {
	db := model.GetDB()
	return &ApplicationCluster{applicationClusterRepo: application_cluster.NewApplicationClusterRepo(db)}
}

func (srv *ApplicationCluster) GetJoinCluster(
	ctx context.Context, id uint64,
) ([]*model.ApplicationClusterJoinModel, error) {
	return srv.applicationClusterRepo.GetJoinCluster(ctx, id)
}

func (srv *ApplicationCluster) GetList(ctx context.Context, id uint64) (
	[]*model.ApplicationClusterModel, error,
) {
	return srv.applicationClusterRepo.GetList(ctx, id)
}

func (srv *ApplicationCluster) GetFirst(ctx context.Context, id uint64) (model.ApplicationClusterModel, error) {
	return srv.applicationClusterRepo.GetFirst(ctx, id)
}

func (srv *ApplicationCluster) Create(
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

func (srv *ApplicationCluster) Close() {
	srv.applicationClusterRepo.Close()
}
