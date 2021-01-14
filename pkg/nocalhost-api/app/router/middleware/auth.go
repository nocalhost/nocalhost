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
	"regexp"

	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/token"
)

// AuthMiddleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the json web token.
		ctx, err := token.ParseRequest(c)
		log.Infof("context is: %+v", ctx)
		if err != nil {
			api.SendResponse(c, errno.ErrTokenInvalid, nil)
			c.Abort()
			return
		}

		if ctx.IsAdmin == 0 && !checkAccessPermission(c.Request.Method, c.Request.RequestURI) {
			api.SendResponse(c, errno.ErrAccessPermissionDenied, nil)
			c.Abort()
			return
		}

		// set uid to context
		c.Set("uid", ctx.Uuid)
		c.Set("userId", ctx.UserID)
		c.Set("isAdmin", ctx.IsAdmin)

		c.Next()
	}
}

func checkAccessPermission(method, path string) bool {
	permissions := map[string]string{
		"/v1/me":                          "GET",
		"/v1/users/[0-9]+":                "PUT",
		"/v1/users/[0-9]+/dev_space_list": "GET",
		"/v1/dev_space/[0-9]+/detail":     "GET",
		"/v1/dev_space/[0-9]+/recreate":   "POST",
		"/v1/application/[0-9]+":          "GET",
		"/v1/plugin/[0-9]+/recreate":      "POST",
		"/v1/plugin/dev_space":            "GET",
		"/v1/plugin/application/[0-9]+/dev_space/[0-9]+/plugin_sync": "PUT",
	}

	for reg, med := range permissions {
		match, _ := regexp.MatchString(reg, path)
		if match && med == method {
			return true
		}
	}

	return false
}
