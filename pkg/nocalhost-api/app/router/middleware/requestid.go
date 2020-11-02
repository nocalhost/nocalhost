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

	"nocalhost/pkg/nocalhost-api/pkg/utils"
)

// RequestID 透传Request-ID，如果没有则生成一个
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for incoming header, use it if exists
		requestID := c.Request.Header.Get(utils.XRequestID)

		// Create request id with UUID
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
