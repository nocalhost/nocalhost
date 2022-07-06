/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"context"
	"github.com/pkg/errors"
	"nocalhost/internal/nocalhost-api/cache"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/cluster"
)

type Cluster struct {
	clusterRepo *cluster.ClusterBaseRepo
}

func NewClusterService() *Cluster {
	db := model.GetDB()
	return &Cluster{
		clusterRepo: cluster.NewClusterRepo(db),
	}
}

func (srv *Cluster) Evict(id uint64) {
	c := cache.Module(cache.CLUSTER)
	_, _ = c.Delete(id)
}

func (srv *Cluster) GetCache(id uint64) (model.ClusterModel, error) {
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

	c.Add(result.ID, cache.OUT_OF_DATE, &result)
	return result, nil
}

func (srv *Cluster) Update(
	ctx context.Context, update map[string]interface{}, clusterId uint64,
) (*model.ClusterModel, error) {
	defer srv.Evict(clusterId)
	return srv.clusterRepo.Update(ctx, update, clusterId)
}

func (srv *Cluster) Delete(ctx context.Context, clusterId uint64) error {
	defer srv.Evict(clusterId)
	return srv.clusterRepo.Delete(ctx, clusterId)
}

func (srv *Cluster) GetAny(ctx context.Context, where map[string]interface{}) ([]*model.ClusterModel, error) {
	return srv.clusterRepo.GetAny(ctx, where)
}

func (srv *Cluster) Create(ctx context.Context, name, kubeconfig, storageClass, server, extraApiServer, clusterInfo string, userId uint64) (model.ClusterModel, error) {
	c := model.ClusterModel{
		Name:           name,
		UserId:         userId,
		Server:         server,
		KubeConfig:     kubeconfig,
		Info:           clusterInfo,
		StorageClass:   storageClass,
		ExtraApiServer: extraApiServer,
	}
	result, err := srv.clusterRepo.Create(ctx, c)
	if err != nil {
		return c, errors.Wrapf(err, "create cluster")
	}
	srv.Evict(result.GetClusterId())
	return result, nil
}

func (srv *Cluster) GetList(ctx context.Context) ([]*model.ClusterList, error) {
	result, _ := srv.clusterRepo.GetList(ctx)
	return result, nil
}

func (srv *Cluster) Get(ctx context.Context, id uint64) (model.ClusterModel, error) {
	result, err := srv.clusterRepo.Get(ctx, id)
	if err != nil {
		return result, errors.Wrapf(err, "get cluster")
	}
	return result, nil
}

func (srv *Cluster) Close() {
	srv.clusterRepo.Close()
}

func (srv *Cluster) DeleteByCreator(ctx context.Context, userid uint64) error {
	return srv.clusterRepo.DeleteByCreator(ctx, userid)
}
