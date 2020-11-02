/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cache

import (
	"time"
)

const (
	// DefaultExpireTime 默认过期时间
	DefaultExpireTime = 60 * time.Second
	// PrefixCacheKey 业务cache key
	PrefixCacheKey = "nocalhost"
)

var Client Driver

// Driver 定义cache驱动接口
type Driver interface {
	Set(key string, val interface{}, expiration time.Duration) error
	Get(key string, val interface{}) error
	MultiSet(valMap map[string]interface{}, expiration time.Duration) error
	MultiGet(keys []string, valueMap interface{}) error
	Del(keys ...string) error
	Incr(key string, step int64) (int64, error)
	Decr(key string, step int64) (int64, error)
}

// Set 数据
func Set(key string, val interface{}, expiration time.Duration) error {
	return Client.Set(key, val, expiration)
}

// Get 数据
func Get(key string, val interface{}) error {
	return Client.Get(key, val)
}

// MultiSet 批量set
func MultiSet(valMap map[string]interface{}, expiration time.Duration) error {
	return Client.MultiSet(valMap, expiration)
}

// MultiGet 批量获取
func MultiGet(keys []string, valueMap interface{}) error {
	return Client.MultiGet(keys, valueMap)
}

// Del 批量删除
func Del(keys ...string) error {
	return Client.Del(keys...)
}

// Incr 自增
func Incr(key string, step int64) (int64, error) {
	return Client.Incr(key, step)
}

// Decr 自减
func Decr(key string, step int64) (int64, error) {
	return Client.Decr(key, step)
}
