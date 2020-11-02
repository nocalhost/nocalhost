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

package redis

import (
	"fmt"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"

	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// RedisClient redis 客户端
var RedisClient *redis.Client

// Nil redis 返回为空
const Nil = redis.Nil

// Init 实例化一个redis client
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

// InitTestRedis 实例化一个可以用于单元测试的redis
func InitTestRedis() {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	// 打开下面命令可以测试链接关闭的情况
	// defer mr.Close()

	RedisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	fmt.Println("mini redis addr:", mr.Addr())
}
