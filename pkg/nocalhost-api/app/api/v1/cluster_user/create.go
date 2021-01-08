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

package cluster_user

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create Create a development environment for application
// @Summary Create a development environment for application
// @Description Create a development environment for application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body cluster_user.ClusterUserCreateRequest true "cluster user info"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/application/{id}/create_space [post]
func Create(c *gin.Context) {
	var req ClusterUserCreateRequest
	defaultNum := uint64(0)
	req.Memory = &defaultNum
	req.Cpu = &defaultNum
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind ApplicationCluster params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	applicationId := cast.ToUint64(c.Param("id"))
	req.ApplicationId = &applicationId
	devSpace := NewDevSpace(req, c, []byte{})
	result, err := devSpace.Create()
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
