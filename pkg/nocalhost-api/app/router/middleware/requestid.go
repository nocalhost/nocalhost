/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package middleware

import (
	"github.com/gin-gonic/gin"

	"nocalhost/pkg/nocalhost-api/pkg/utils"
)

// RequestID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for incoming header, use it if exists
		requestID := c.Request.Header.Get(utils.XRequestID)

		// Initial request id with UUID
		if requestID == "" {
			requestID = utils.GenRequestID()
		}

		// Expose it for use in the application
		c.Set(utils.XRequestID, requestID)

		// Set X-Request-ID header
		c.Writer.Header().Set(utils.XRequestID, requestID)
		c.Next()
	}
}
