/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"github.com/qiniu/api.v7/storage"
	"github.com/spf13/viper"
	"io"
	"k8s.io/apimachinery/pkg/util/validation"
	"math/rand"
	"net"
	"nocalhost/pkg/nocalhost-api/pkg/constvar"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/teris-io/shortid"
	tnet "github.com/toolkits/net"
)

var emailReg = "(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9]))\\.){3}(?:(2(5[0-5]|[0-4][0-9])|1[0-9][0-9]|[1-9]?[0-9])|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])"

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

func IsEmail(email string) bool {

	match, err := regexp.Match(emailReg, []byte(strings.ToLower(email)))
	return err == nil && match
}

// GenShortID
func GenShortID() (string, error) {
	return shortid.Generate()
}

// XRequestID
var XRequestID = "X-Request-ID"

// GenRequestID eg: 76d27e8c-a80e-48c8-ad20-e5562e0f67e4
func GenRequestID() string {
	u, _ := uuid.NewRandom()
	return u.String()
}

// GetRequestID
func GetRequestID(c *gin.Context) string {
	v, ok := c.Get(XRequestID)
	if !ok {
		return ""
	}
	if requestID, ok := v.(string); ok {
		return requestID
	}
	return ""
}

var (
	once     sync.Once
	clientIP = "127.0.0.1"
)

// GetLocalIP
func GetLocalIP() string {
	once.Do(
		func() {
			ips, _ := tnet.IntranetIP()
			if len(ips) > 0 {
				clientIP = ips[0]
			} else {
				clientIP = "127.0.0.1"
			}
		},
	)
	return clientIP
}

// GetBytes interface to byte
func GetBytes(key interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(key)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Md5
func Md5(str string) (string, error) {
	h := md5.New()

	_, err := io.WriteString(h, str)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// RandomStr
func RandomStr(n int) string {
	var r = rand.New(rand.NewSource(time.Now().UnixNano()))
	const pattern = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz"

	salt := make([]byte, 0, n)
	l := len(pattern)

	for i := 0; i < n; i++ {
		p := r.Intn(l)
		salt = append(salt, pattern[p])
	}

	return string(salt)
}

// RegexpReplace ...
func RegexpReplace(reg, src, temp string) string {
	result := []byte{}
	pattern := regexp.MustCompile(reg)
	for _, submatches := range pattern.FindAllStringSubmatchIndex(src, -1) {
		result = pattern.ExpandString(result, temp, src, submatches)
	}
	return string(result)
}

// GetRealIP get user real ip
func GetRealIP(ctx *gin.Context) (ip string) {
	var header = ctx.Request.Header
	var index int
	if ip = header.Get("X-Forwarded-For"); ip != "" {
		index = strings.IndexByte(ip, ',')
		if index < 0 {
			return ip
		}
		if ip = ip[:index]; ip != "" {
			return ip
		}
	}
	if ip = header.Get("X-Real-Ip"); ip != "" {
		index = strings.IndexByte(ip, ',')
		if index < 0 {
			return ip
		}
		if ip = ip[:index]; ip != "" {
			return ip
		}
	}
	if ip = header.Get("Proxy-Forwarded-For"); ip != "" {
		index = strings.IndexByte(ip, ',')
		if index < 0 {
			return ip
		}
		if ip = ip[:index]; ip != "" {
			return ip
		}
	}
	ip, _, _ = net.SplitHostPort(ctx.Request.RemoteAddr)
	return ip
}

// is match dns label
func ReplaceDNS1123(name string) string {
	var invalidDNS1123Characters = regexp.MustCompile("[^-a-z0-9]+")
	name = strings.ToLower(name)
	name = invalidDNS1123Characters.ReplaceAllString(name, "-")
	if len(name) > validation.DNS1123LabelMaxLength {
		name = name[0:validation.DNS1123LabelMaxLength]
	}
	return strings.Trim(name, "-")
}
