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

package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/util/validation"
	"math/rand"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/teris-io/shortid"
	tnet "github.com/toolkits/net"
)

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
	once.Do(func() {
		ips, _ := tnet.IntranetIP()
		if len(ips) > 0 {
			clientIP = ips[0]
		} else {
			clientIP = "127.0.0.1"
		}
	})
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
