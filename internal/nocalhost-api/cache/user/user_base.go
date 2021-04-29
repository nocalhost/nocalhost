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

package user

import (
	"fmt"
	"time"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/pkg/nocalhost-api/pkg/cache"
	"nocalhost/pkg/nocalhost-api/pkg/redis"
)

const (
	// PrefixUserBaseCacheKey cache prefix
	PrefixUserBaseCacheKey = "user:cache:%d"
	// DefaultExpireTime
	DefaultExpireTime = time.Hour * 24
)

// Cache cache
type Cache struct {
	cache cache.Driver
}

// NewUserCache
func NewUserCache() *Cache {
	encoding := cache.JSONEncoding{}
	cachePrefix := cache.PrefixCacheKey
	return &Cache{
		cache: cache.NewRedisCache(redis.RedisClient, cachePrefix, encoding, func() interface{} {
			return &model.UserBaseModel{}
		}),
	}
}

// GetUserBaseCacheKey
func (u *Cache) GetUserBaseCacheKey(userID uint64) string {
	return fmt.Sprintf(cache.PrefixCacheKey+":"+PrefixUserBaseCacheKey, userID)
}

// SetUserBaseCache
func (u *Cache) SetUserBaseCache(userID uint64, user *model.UserBaseModel) error {
	if user == nil || user.ID == 0 {
		return nil
	}
	cacheKey := fmt.Sprintf(PrefixUserBaseCacheKey, userID)
	err := u.cache.Set(cacheKey, user, DefaultExpireTime)
	if err != nil {
		return err
	}
	return nil
}

// GetUserBaseCache
func (u *Cache) GetUserBaseCache(userID uint64) (data *model.UserBaseModel, err error) {
	cacheKey := fmt.Sprintf(PrefixUserBaseCacheKey, userID)
	err = u.cache.Get(cacheKey, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// MultiGetUserBaseCache
func (u *Cache) MultiGetUserBaseCache(userIDs []uint64) (map[string]*model.UserBaseModel, error) {
	var keys []string
	for _, v := range userIDs {
		cacheKey := fmt.Sprintf(PrefixUserBaseCacheKey, v)
		keys = append(keys, cacheKey)
	}

	userMap := make(map[string]*model.UserBaseModel)
	err := u.cache.MultiGet(keys, userMap)
	if err != nil {
		return nil, err
	}
	return userMap, nil
}

// DelUserBaseCache
func (u *Cache) DelUserBaseCache(userID uint64) error {
	cacheKey := fmt.Sprintf(PrefixUserBaseCacheKey, userID)
	err := u.cache.Del(cacheKey)
	if err != nil {
		return err
	}
	return nil
}
