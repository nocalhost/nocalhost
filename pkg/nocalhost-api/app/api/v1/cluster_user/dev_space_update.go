/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/manager"
)

type DevSpaceUpdate struct {
	Params DevSpaceRequest
	c      *gin.Context
}

func (d *DevSpaceUpdate) UpdateVirtualCluster(cu model.ClusterUserModel) error {
	if d.Params.VirtualCluster == nil {
		return errors.New("virtual cluster is nil")
	}
	v := d.Params.VirtualCluster

	space, err := service.Svc.ClusterUser().GetFirst(d.c, model.ClusterUserModel{ID: cu.ID})
	if err != nil {
		return err
	}
	cluster, err := service.Svc.ClusterSvc().Get(d.c, space.ClusterId)
	if err != nil {
		return err
	}

	f := manager.VClusterSharedManagerFactory
	m, err := f.Manager(cluster.GetKubeConfig())
	if err != nil {
		return err
	}
	return m.Update(cu.SpaceName, space.Namespace, cluster.GetClusterName(), v)
}

func NewDecSpaceUpdater(request DevSpaceRequest, c *gin.Context) *DevSpaceUpdate {
	return &DevSpaceUpdate{
		Params: request,
		c:      c,
	}
}
