/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	helmv1alpha1 "nocalhost/internal/nocalhost-dep/controllers/vcluster/api/v1alpha1"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
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

	space, err := service.Svc.ClusterUser().GetFirst(d.c, cu)
	if err != nil {
		return err
	}
	cluster, err := service.Svc.ClusterSvc().Get(d.c, space.ClusterId)
	if err != nil {
		return err
	}

	goClient, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		return err
	}

	obj, err := goClient.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    "helm.nocalhost.dev",
		Version:  "v1alpha1",
		Resource: "virtualclusters",
	}).Namespace(space.Namespace).Get(context.TODO(), space.SpaceName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	vc := &helmv1alpha1.VirtualCluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(
		obj.UnstructuredContent(), vc); err != nil {
		return errors.WithStack(err)
	}

	vc.SetValues(v.Values)
	vc.SetChartVersion(v.Version)
	annotations := map[string]string{
		helmv1alpha1.ServiceTypeKey: string(v.ServiceType),
	}
	vc.SetAnnotations(annotations)
	vc.SetManagedFields(nil)

	_, err = goClient.Apply(vc)
	return err
}

func NewDecSpaceUpdater(request DevSpaceRequest, c *gin.Context) *DevSpaceUpdate {
	return &DevSpaceUpdate{
		Params: request,
		c:      c,
	}
}
