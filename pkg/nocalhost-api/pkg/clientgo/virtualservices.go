/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgo

import (
	"context"

	"github.com/pkg/errors"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *GoClient) GetVirtualService(namespace, name string) (*v1alpha3.VirtualService, error) {
	resource, err := c.istioClient.
		NetworkingV1alpha3().
		VirtualServices(namespace).
		Get(context.TODO(), name, metav1.GetOptions{})
	return resource, errors.WithStack(err)
}

// GetVirtualServiceList get a list of virtualservices.
func (c *GoClient) GetVirtualServiceList(namespace string) (*v1alpha3.VirtualServiceList, error) {
	resources, err := c.istioClient.
		NetworkingV1alpha3().
		VirtualServices(namespace).
		List(context.TODO(), metav1.ListOptions{})
	return resources, errors.WithStack(err)
}

func (c *GoClient) CreateVirtualService(namespace string,
	vs *v1alpha3.VirtualService) (*v1alpha3.VirtualService, error) {
	resource, err := c.istioClient.
		NetworkingV1alpha3().
		VirtualServices(namespace).
		Create(context.TODO(), vs, metav1.CreateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) UpdateVirtualService(namespace string,
	vs *v1alpha3.VirtualService) (*v1alpha3.VirtualService, error) {
	resource, err := c.istioClient.
		NetworkingV1alpha3().
		VirtualServices(namespace).
		Update(context.TODO(), vs, metav1.UpdateOptions{})
	return resource, errors.WithStack(err)
}

func (c *GoClient) DeleteVirtualService(namespace, name string) error {
	return errors.WithStack(c.istioClient.
		NetworkingV1alpha3().
		VirtualServices(namespace).
		Delete(context.TODO(), name, metav1.DeleteOptions{}))
}
