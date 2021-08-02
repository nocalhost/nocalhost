/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package redis

import (
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

// Lock
type Lock struct {
	key         string
	redisClient *redis.Client
	timeout     time.Duration
}

// NewLock
func NewLock(conn *redis.Client, key string, defaultTimeout time.Duration) *Lock {
	return &Lock{
		key:         key,
		redisClient: conn,
		timeout:     defaultTimeout,
	}
}

// Lock
func (l *Lock) Lock(token string) (bool, error) {
	ok, err := l.redisClient.SetNX(l.GetKey(), token, l.timeout).Result()
	if err == redis.Nil {
		err = nil
	}
	return ok, err
}

// Unlock
func (l *Lock) Unlock(token string) error {
	script := "if redis.call('get',KEYS[1]) == ARGV[1] then return redis.call('del',KEYS[1]) else return 0 end"
	_, err := l.redisClient.Eval(script, []string{l.GetKey()}, token).Result()
	if err != nil {
		return err
	}
	return nil
}

// GetKey
func (l *Lock) GetKey() string {
	keyPrefix := viper.GetString("name")
	lockKey := "redis:lock"
	return strings.Join([]string{keyPrefix, lockKey, l.key}, ":")
}

// GenToken
func (l *Lock) GenToken() string {
	u, _ := uuid.NewRandom()
	return u.String()
}
