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

package middleware

import (
	"github.com/gin-gonic/gin"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"regexp"
	"strings"
)

// PermissionMiddleware
func PermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, err := c.Get("isAdmin")
		if !err {
			api.SendResponse(c, errno.ErrLostPermissionFlag, nil)
			c.Abort()
			return
		}

		if isAdmin.(uint64) != 1 && !whiteList(c.Request.Method, c.Request.RequestURI) {
			api.SendResponse(c, errno.ErrPermissionDenied, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func whiteList(method, path string) bool {
	permissions := map[string]string{
		"/v1/users/[0-9]+":                "PUT",
		"/v1/users/[0-9]+/dev_space_list": "GET",
		"/v1/users/[0-9]+/applications":   "GET",
		"/v1/users/[0-9]+/dev_spaces":     "GET",
		"/v1/dev_space/[0-9]+/detail":     "GET",
		"/v1/dev_space/[0-9]+/recreate":   "POST",
		"/v1/application/[0-9]+":          "GET,PUT,DELETE",
		"/v1/nocalhost/templates":         "GET",
		"/v1/dev_space":                   "GET",
		"/v1/application":                 "GET,POST",
	}

	for reg, med := range permissions {
		match, _ := regexp.MatchString(reg, path)
		if match && strings.Contains(med, method) {
			return true
		}
	}

	return false
}
