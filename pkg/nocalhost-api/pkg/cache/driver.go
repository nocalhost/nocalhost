/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cache

import (
	"time"
)

const (
	// DefaultExpireTime
	DefaultExpireTime = 60 * time.Second
	// PrefixCacheKey cache key
	PrefixCacheKey = "nocalhost"
)

var Client Driver

// Driver
type Driver interface {
	Set(key string, val interface{}, expiration time.Duration) error
	Get(key string, val interface{}) error
	MultiSet(valMap map[string]interface{}, expiration time.Duration) error
	MultiGet(keys []string, valueMap interface{}) error
	Del(keys ...string) error
	Incr(key string, step int64) (int64, error)
	Decr(key string, step int64) (int64, error)
}

// Set
func Set(key string, val interface{}, expiration time.Duration) error {
	return Client.Set(key, val, expiration)
}

// Get
func Get(key string, val interface{}) error {
	return Client.Get(key, val)
}

// MultiSet
func MultiSet(valMap map[string]interface{}, expiration time.Duration) error {
	return Client.MultiSet(valMap, expiration)
}

// MultiGet
func MultiGet(keys []string, valueMap interface{}) error {
	return Client.MultiGet(keys, valueMap)
}

// Del
func Del(keys ...string) error {
	return Client.Del(keys...)
}

// Incr
func Incr(key string, step int64) (int64, error) {
	return Client.Incr(key, step)
}

// Decr
func Decr(key string, step int64) (int64, error) {
	return Client.Decr(key, step)
}
