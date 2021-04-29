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

package cache

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	"nocalhost/pkg/nocalhost-api/pkg/log"
)

type memoryCache struct {
	client    *sync.Map
	KeyPrefix string
	encoding  Encoding
}

// NewMemoryCache
func NewMemoryCache(keyPrefix string, encoding Encoding) Driver {
	return &memoryCache{
		client:    &sync.Map{},
		KeyPrefix: keyPrefix,
		encoding:  encoding,
	}
}

// item
type itemWithTTL struct {
	expires int64
	value   interface{}
}

// newItem
func newItem(value interface{}, expires time.Duration) itemWithTTL {
	expires64 := int64(expires)
	if expires > 0 {
		expires64 = time.Now().Unix() + expires64
	}
	return itemWithTTL{
		value:   value,
		expires: expires64,
	}
}

// getValue
func getValue(item interface{}, ok bool) (interface{}, bool) {
	if !ok {
		return nil, false
	}

	var itemObj itemWithTTL
	if itemObj, ok = item.(itemWithTTL); !ok {
		return nil, false
	}

	if itemObj.expires > 0 && itemObj.expires < time.Now().Unix() {
		return nil, false
	}

	return itemObj.value, true
}

// Set data
func (m memoryCache) Set(key string, val interface{}, expiration time.Duration) error {
	cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
	if err != nil {
		return errors.Wrapf(err, "build cache key err, key is %+v", key)
	}
	m.client.Store(cacheKey, newItem(val, expiration))
	return nil
}

// Get data
func (m memoryCache) Get(key string, val interface{}) error {
	cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
	if err != nil {
		return errors.Wrapf(err, "build cache key err, key is %+v", key)
	}
	val, ok := getValue(m.client.Load(cacheKey))
	if !ok {
		return errors.New("memory get value err")
	}
	return nil
}

// MultiSet
func (m memoryCache) MultiSet(valMap map[string]interface{}, expiration time.Duration) error {
	panic("implement me")
}

// MultiGet
func (m memoryCache) MultiGet(keys []string, val interface{}) error {
	panic("implement me")
}

// Del
func (m memoryCache) Del(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		cacheKey, err := BuildCacheKey(m.KeyPrefix, key)
		if err != nil {
			log.Warnf("build cache key err: %+v, key is %+v", err, key)
			continue
		}
		m.client.Delete(cacheKey)
	}
	return nil
}

// Incr
func (m memoryCache) Incr(key string, step int64) (int64, error) {
	panic("implement me")
}

// Decr
func (m memoryCache) Decr(key string, step int64) (int64, error) {
	panic("implement me")
}
