/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package cache

import (
	"errors"
	"strings"
)

// BuildCacheKey
func BuildCacheKey(keyPrefix string, key string) (cacheKey string, err error) {
	if key == "" {
		return "", errors.New("[cache] key should not be empty")
	}

	cacheKey, err = strings.Join([]string{keyPrefix, key}, ":"), nil
	return
}
