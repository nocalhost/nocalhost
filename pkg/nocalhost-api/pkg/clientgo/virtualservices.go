/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
