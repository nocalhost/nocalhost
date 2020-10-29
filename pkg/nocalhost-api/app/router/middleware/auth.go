package middleware

import (
	"github.com/gin-gonic/gin"

	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/token"
)

// AuthMiddleware 认证中间件
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

		// set uid to context
		c.Set("uid", ctx.Uuid)
		c.Set("userId", ctx.UserID)

		c.Next()
	}
}
