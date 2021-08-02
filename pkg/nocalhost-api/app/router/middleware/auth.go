/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package middleware

import (
	"github.com/gin-gonic/gin"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/token"
)

// AuthMiddleware
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse the json web token.
		ctx, err := token.ParseRequest(c)
		if err != nil {
			api.SendResponse(c, errno.ErrTokenInvalid, nil)
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
