/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package middleware

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// PermissionMiddleware
func PermissionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := IsAdmin(c)
		if err != nil {
			api.SendResponse(c, err, nil)
			c.Abort()
			return
		}

		if !admin && !whiteList(c.Request.Method, c.Request.RequestURI) {
			api.SendResponse(c, errno.ErrPermissionDenied, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func whiteList(method, path string) bool {
	permissions := map[string]string{
		"/v1/users":                          "GET",
		"/v1/users/[0-9]+":                   "PUT",
		"/v1/users/[0-9]+/dev_space_list":    "GET",
		"/v1/users/[0-9]+/applications":      "GET",
		"/v1/users/[0-9]+/dev_spaces":        "GET",
		"/v1/dev_space/[0-9]+/detail":        "GET",
		"/v1/dev_space/[0-9]+/recreate":      "POST",
		"/v1/application/[0-9]+":             "GET,PUT,DELETE",
		"/v1/nocalhost/templates":            "GET",
		"/v1/nocalhost/version/upgrade_info": "GET",
		"/v1/dev_space":                      "GET,POST",

		"/v1/application": "GET,POST",

		"/v1/cluster":                      "POST,GET",
		"/v1/cluster/[0-9]+":               "PUT,DELETE",
		"/v1/cluster/[0-9]+/storage_class": "PUT,DELETE",
		"/v1/cluster/[0-9]+/gen_namespace": "GET",
		"/v1/cluster/[0-9]+/migrate":       "POST",

		"/v1/dev_space/[0-9]+/update_resource_limit": "PUT",
		"/v1/dev_space/[0-9]+":                       "PUT,DELETE",

		"/v2/dev_space":         "GET",
		"/v2/dev_space/cluster": "GET",
		"/v2/dev_space/share":   "POST",
		"/v2/dev_space/unshare": "POST",
	}

	for reg, med := range permissions {
		match, _ := regexp.MatchString(reg, path)
		if match && strings.Contains(med, method) {
			return true
		}
	}

	return false
}

func IsAdmin(c *gin.Context) (bool, error) {
	id, ok := c.Get("isAdmin")
	if !ok {
		return false, errno.ErrLostPermissionFlag
	}
	return id.(uint64) == 1, nil
}
