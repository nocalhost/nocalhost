/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package redis

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// IdAlloc
type IdAlloc struct {
	key         string
	redisClient *redis.Client
}

// New
func New(conn *redis.Client, key string, defaultTimeout time.Duration) *Lock {
	return &Lock{
		key:         key,
		redisClient: conn,
		timeout:     defaultTimeout,
	}
}

// GetNewID
func (ia *IdAlloc) GetNewID(step int64) (int64, error) {
	key := ia.GetKey()
	id, err := ia.redisClient.IncrBy(key, step).Result()
	if err != nil {
		return 0, errors.Wrapf(err, "redis incr err, key: %s", key)
	}
	return id, nil
}

// GetCurrentID
func (ia *IdAlloc) GetCurrentID() (int64, error) {
	key := ia.GetKey()
	ret, err := ia.redisClient.Get(key).Result()
	if err != nil {
		return 0, errors.Wrapf(err, "redis get err, key: %s", key)
	}
	id, err := strconv.Atoi(ret)
	if err != nil {
		return 0, errors.Wrap(err, "str convert err")
	}
	return int64(id), nil
}

// GetKey
func (ia *IdAlloc) GetKey() string {
	keyPrefix := viper.GetString("name")
	lockKey := "idalloc"
	return strings.Join([]string{keyPrefix, lockKey, ia.key}, ":")
}
