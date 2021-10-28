/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgo

import (
	"context"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (c *GoClient) ApplyForce(object interface{}) (*unstructured.Unstructured, error) {

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

	force := true
	result, err := client.Patch(context.TODO(),
		obj.GetName(),
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: Api, Force: &force})
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

func (c *GoClient) CheckIstio() (bool, error) {
	if _, err := c.DynamicClient.Resource(schema.GroupVersionResource{
		Group:    "apiregistration.k8s.io",
		Version:  "v1",
		Resource: "apiservices",
	}).Get(context.TODO(), "v1alpha3.networking.istio.io", metav1.GetOptions{}); err != nil {
		return false, err
	}
	return true, nil
}
