/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
)

func (c *ClientGoUtils) GetResourceInfo(resourceType string, resourceName string) (*resource.Info, error) {
	r := c.NewFactory().NewBuilder().
		Unstructured().
		LabelSelector(labels.SelectorFromSet(c.labels).String()).
		NamespaceParam(c.namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(true, []string{resourceType, resourceName}...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()

	if err := r.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	infos, err := r.Infos()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(infos) > 1 {
		return nil, errors.New(fmt.Sprintf("ResouceInfo is not 1(but %d)?", len(infos)))
	}
	return infos[0], nil
}

func (c *ClientGoUtils) Get(resourceType string, resourceName string) (*runtime.Object, error) {
	info, err := c.GetResourceInfo(resourceType, resourceName)
	if err != nil {
		return nil, err
	}
	return &info.Object, nil
}

func (c *ClientGoUtils) ListResourceInfo(resourceType string) ([]*resource.Info, error) {
	r := c.NewFactory().NewBuilder().
		Unstructured().
		LabelSelector(labels.SelectorFromSet(c.labels).String()).
		NamespaceParam(c.namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(true, []string{resourceType}...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()

	if err := r.Err(); err != nil {
		return nil, errors.WithStack(err)
	}

	infos, err := r.Infos()
	return infos, errors.WithStack(err)
}

func (c *ClientGoUtils) GetUnstructured(resourceType string, resourceName string) (*unstructured.Unstructured, error) {
	obj, err := c.Get(resourceType, resourceName)
	if err != nil {
		return nil, err
	}

	us, ok := (*obj).(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("Fail to assert")
	}
	return us, nil
}

func (c *ClientGoUtils) GetUnstructuredFromString(str string) (*unstructured.Unstructured, error) {
	infos, err := c.GetResourceInfoFromString(str, true)
	if err != nil {
		return nil, err
	}

	if len(infos) != 1 {
		return nil, errors.New(fmt.Sprintf("%d infos found, not 1?", len(infos)))
	}

	obj, ok := infos[0].Object.(*unstructured.Unstructured)
	if !ok {
		return nil, errors.New("can not convert to unstructured")
	}
	return obj, err
}
