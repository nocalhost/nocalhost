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

package api

import (
	"github.com/spf13/viper"
	"net/http"
	"nocalhost/pkg/nocalhost-api/napp"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"os"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// Response api
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// SendResponse
func SendResponse(c *gin.Context, err error, data interface{}) {
	code, message := errno.DecodeErr(err)

	// always return http.StatusOK
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// GetUserID
func GetUserID(c *gin.Context) uint64 {
	if c == nil {
		return 0
	}

	// The uid must be named the same as the uid in middleware/auth
	if v, exists := c.Get("uid"); exists {
		uid, ok := v.(uint64)
		if !ok {
			return 0
		}

		return uid
	}
	return 0
}

// RouteNotFound
func RouteNotFound(c *gin.Context) {
	//c.String(http.StatusNotFound, "the route not found")
	SendResponse(c, errno.RouterNotFound, nil)
	return
}

// getHostname
func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		name = "unknown"
	}
	return name
}

// healthCheckResponse
type healthCheckResponse struct {
	Status   string `json:"status"`
	Hostname string `json:"hostname"`
}

// HealthCheck will return OK if the underlying BoltDB is healthy. At least healthy enough for demoing purposes.
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, healthCheckResponse{Status: "UP", Hostname: getHostname()})
}

// global handle 500 error
func Recover(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {

			log.Errorf("panic: %v\n", r)
			// debug
			if viper.GetString("app.run_mode") == napp.ModeDebug {
				debug.PrintStack()
			}
			SendResponse(c, errno.InternalServerError, nil)
			return
		}
	}()
	c.Next()
}
