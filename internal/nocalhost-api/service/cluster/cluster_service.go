/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"context"
	"nocalhost/internal/nocalhost-api/cache"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster"
	"time"

	"github.com/pkg/errors"
)

type ClusterService interface {
	Create(
		ctx context.Context, name, kubeconfig, storageClass, server, clusterInfo string, userId uint64,
	) (model.ClusterModel, error)
	Get(ctx context.Context, id uint64) (model.ClusterModel, error)
	Delete(ctx context.Context, clusterId uint64) error
	GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error)
	Update(ctx context.Context, update map[string]interface{}, clusterId uint64) (*model.ClusterModel, error)
	GetList(ctx context.Context) ([]*model.ClusterList, error)

	GetCache(id uint64) (model.ClusterModel, error)
	Close()
	DeleteByCreator(ctx context.Context, id uint64) error
}

type clusterService struct {
	clusterRepo cluster.ClusterRepo
}

func NewClusterService() ClusterService {
	db := model.GetDB()
	return &clusterService{
		clusterRepo: cluster.NewClusterRepo(db),
	}
}

func (srv *clusterService) Evict(id uint64) {
	c := cache.Module(cache.CLUSTER)
	_, _ = c.Delete(id)
}

func (srv *clusterService) GetCache(id uint64) (model.ClusterModel, error) {
	c := cache.Module(cache.CLUSTER)
	value, err := c.Value(id)
	if err == nil {
		clusterModel := value.Data().(*model.ClusterModel)
		return *clusterModel, nil
	}

	result, err := srv.Get(context.TODO(), id)
	if err != nil {
		return result, errors.Wrapf(err, "get cluster")
	}

	c.Add(result.ID, time.Hour, &result)
	return result, nil
}

func (srv *clusterService) Update(
	ctx context.Context, update map[string]interface{}, clusterId uint64,
) (*model.ClusterModel, error) {
	defer srv.Evict(clusterId)
	return srv.clusterRepo.Update(ctx, update, clusterId)
}

func (srv *clusterService) Delete(ctx context.Context, clusterId uint64) error {
	defer srv.Evict(clusterId)
	return srv.clusterRepo.Delete(ctx, clusterId)
}

func (srv *clusterService) GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error) {
	return srv.clusterRepo.GetAny(ctx, where)
}

func (srv *clusterService) Create(
	ctx context.Context, name, kubeconfig, storageClass, server, clusterInfo string, userId uint64,
) (model.ClusterModel, error) {
	c := model.ClusterModel{
		Name:         name,
		UserId:       userId,
		Server:       server,
		KubeConfig:   kubeconfig,
		Info:         clusterInfo,
		StorageClass: storageClass,
	}
	result, err := srv.clusterRepo.Create(ctx, c)
	if err != nil {
		return c, errors.Wrapf(err, "create cluster")
	}
	srv.Evict(result.GetClusterId())
	return result, nil
}

func (srv *clusterService) GetList(ctx context.Context) ([]*model.ClusterList, error) {
	result, _ := srv.clusterRepo.GetList(ctx)
	return result, nil
}

func (srv *clusterService) Get(ctx context.Context, id uint64) (model.ClusterModel, error) {
	result, err := srv.clusterRepo.Get(ctx, id)
	if err != nil {
		return result, errors.Wrapf(err, "get cluster")
	}
	return result, nil
}

func (srv *clusterService) Close() {
	srv.clusterRepo.Close()
}

func (srv *clusterService) DeleteByCreator(ctx context.Context, userid uint64) error {
	return srv.clusterRepo.DeleteByCreator(ctx, userid)
}
