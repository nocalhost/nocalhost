/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cache

import (
	"sync"
	"time"
)

type Cache struct {
	cache  sync.Map
	expire time.Duration
}

func NewCache(duration time.Duration) *Cache {
	return &Cache{expire: duration}
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	return c.cache.Load(key)
}

func (c *Cache) Set(key interface{}, value interface{}) {
	c.cache.Store(key, value)
	time.AfterFunc(c.expire, func() {
		c.cache.Delete(key)
	})
}
