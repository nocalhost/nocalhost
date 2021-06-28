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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func (c *GoClient) Apply(object interface{}) (*unstructured.Unstructured, error) {

	obj := &unstructured.Unstructured{}
	var err error
	obj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := c.buildDynamicResourceClient(obj)
	if err != nil {
		return nil, err
	}

	result, err := client.Patch(context.TODO(),
		obj.GetName(),
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: Api})
	return result, errors.WithStack(err)
}

func (c *GoClient) Delete(object interface{}) error {

	obj := &unstructured.Unstructured{}
	var err error
	obj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return errors.WithStack(err)
	}

	client, err := c.buildDynamicResourceClient(obj)
	if err != nil {
		return err
	}

	return errors.WithStack(client.Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{}))
}

func (c *GoClient) buildDynamicResourceClient(obj *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gvk := obj.GroupVersionKind()

	restMapper, err := c.GetDiscoveryRESTMapper()
	if err != nil {
		return nil, err
	}

	restMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if restMapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(metav1.NamespaceDefault)
		}
		return c.DynamicClient.Resource(restMapping.Resource).Namespace(obj.GetNamespace()), nil
	}

	return c.DynamicClient.Resource(restMapping.Resource), nil
}
