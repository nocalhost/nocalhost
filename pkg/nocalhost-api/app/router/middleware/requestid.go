/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
