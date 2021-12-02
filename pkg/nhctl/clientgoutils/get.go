/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgoutils

import (
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

func (c *ClientGoUtils) Get(resourceType string, resourceName string) (*runtime.Object, error) {
	r := c.NewFactory().NewBuilder().
		Unstructured().
		NamespaceParam(c.namespace).DefaultNamespace().
		ResourceTypeOrNameArgs(true, []string{resourceType, resourceName}...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()

	if err := r.Err(); err != nil {
		return nil, errors.Wrap(err, "")
	}

	infos, err := r.Infos()
	if err != nil {
		return nil, errors.Wrap(err, "")
	}

	objs := make([]runtime.Object, len(infos))
	for ix := range infos {
		objs[ix] = infos[ix].Object
	}

	if len(objs) == 0 {
		return nil, errors.New(fmt.Sprintf("Resource %s(%s) not found", resourceName, resourceType))
	}

	return &objs[0], nil
}

func (c *ClientGoUtils) GetUnstructuredMap(resourceType string, resourceName string) (map[string]interface{}, error) {
	obj, err := c.Get(resourceType, resourceName)
	if err != nil {
		return nil, err
	}

	var unstructuredObj map[string]interface{}
	if unstructuredObj, err = runtime.DefaultUnstructuredConverter.ToUnstructured(obj); err != nil {
		return nil, errors.WithStack(err)
	}
	return unstructuredObj, nil
}
