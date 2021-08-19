/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package clientgo

import (
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/restmapper"
)

func (c *GoClient) GetDiscoveryRESTMapper() (*restmapper.DeferredDiscoveryRESTMapper, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mapper != nil {
		return c.mapper, nil
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(c.restConfig)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	c.mapper = restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))
	return c.mapper, nil
}

func (c *GoClient) GetDiscoveryClient() (*discovery.DiscoveryClient, error) {
	return discovery.NewDiscoveryClientForConfig(c.restConfig)
}
