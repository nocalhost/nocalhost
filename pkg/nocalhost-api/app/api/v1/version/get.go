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

package version

import (
	"github.com/gin-gonic/gin"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// Get Get api server version
// @Summary Get api server version
// @Description Get api server version
// @Tags Version
// @Accept  json
// @Produce  json
// @Success 200 {string} json "{"code":0,"message":"OK","data":{"version":"","commit_id":"","branch":""}}"
// @Router /v1/version [get]
func Get(c *gin.Context) {
	version := map[string]string{
		"version":   global.Version,
		"commit_id": global.CommitId,
		"branch":    global.Branch,
	}
	api.SendResponse(c, errno.OK, version)
}
