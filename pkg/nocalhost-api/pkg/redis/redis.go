/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package redis

import (
	"fmt"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"

	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// RedisClient
var RedisClient *redis.Client

// Nil
const Nil = redis.Nil

func Init() *redis.Client {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.addr"),
		Password:     viper.GetString("redis.password"),
		DB:           viper.GetInt("redis.db"),
		DialTimeout:  viper.GetDuration("redis.dial_timeout"),
		ReadTimeout:  viper.GetDuration("redis.read_timeout"),
		WriteTimeout: viper.GetDuration("redis.write_timeout"),
		PoolSize:     viper.GetInt("redis.pool_size"),
		PoolTimeout:  viper.GetDuration("redis.pool_timeout"),
	})

	fmt.Println("redis addr:", viper.GetString("redis.addr"))

	_, err := RedisClient.Ping().Result()
	if err != nil {
		log.Errorf("[redis] redis ping err: %+v", err)
		panic(err)
	}
	return RedisClient
}

// InitTestRedis
func InitTestRedis() {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	// defer mr.Close()

	RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	fmt.Println("mini redis addr:", mr.Addr())
}
