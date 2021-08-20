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
	cache         sync.Map
	cancelFuncMap sync.Map
	expire        time.Duration
	lock          sync.Mutex
}

func NewCache(duration time.Duration) *Cache {
	return &Cache{expire: duration}
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.cache.Load(key)
}

func (c *Cache) Set(key interface{}, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	// delete key already exist
	c.cache.Delete(key)
	// stop timer function, because value already updated, very important, attention here !!!
	if cancelFunc, loaded := c.cancelFuncMap.LoadAndDelete(key); loaded && cancelFunc != nil {
		cancelFunc.(*time.Timer).Stop()
	}
	c.cache.Store(key, value)
	c.cancelFuncMap.Store(key, time.AfterFunc(c.expire, func() { c.cache.Delete(key) }))
}
