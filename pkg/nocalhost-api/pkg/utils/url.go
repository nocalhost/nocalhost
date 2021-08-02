/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package utils

import (
	"strings"

	"github.com/qiniu/api.v7/storage"
	"github.com/spf13/viper"

	"nocalhost/pkg/nocalhost-api/pkg/constvar"
)

// GetDefaultAvatarURL
func GetDefaultAvatarURL() string {
	return GetQiNiuPublicAccessURL(constvar.DefaultAvatar)
}

// GetAvatarURL user's avatar, if empty, use default avatar
func GetAvatarURL(key string) string {
	if key == "" {
		return GetDefaultAvatarURL()
	}
	if strings.HasPrefix(key, "https://") {
		return key
	}
	return GetQiNiuPublicAccessURL(key)
}

// GetQiNiuPublicAccessURL
func GetQiNiuPublicAccessURL(path string) string {
	domain := viper.GetString("qiniu.cdn_url")
	key := strings.TrimPrefix(path, "/")

	publicAccessURL := storage.MakePublicURL(domain, key)

	return publicAccessURL
}
