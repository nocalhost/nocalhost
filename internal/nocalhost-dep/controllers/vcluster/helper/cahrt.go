/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package helper

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"helm.sh/helm/v3/pkg/chart"
)

const (
	defaultLRUSize = 10
)

var ChartCache = newChartCache()

type chartCache struct {
	mu  sync.Mutex
	lru *lru.Cache
}

func (c *chartCache) Get(key string) (*chart.Chart, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.lru.Get(key); ok {
		return v.(*chart.Chart), true
	}
	return nil, false
}

func (c *chartCache) Add(key string, value *chart.Chart) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lru.Add(key, value)
}

func newChartCache() *chartCache {
	cache, _ := lru.New(defaultLRUSize)
	return &chartCache{
		lru: cache,
	}
}
