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
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

type DevSpaceRequest struct {
	KubeConfig string `json:"kubeconfig"`
}

// @Summary Update dev space
// @Description Update dev space
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "devspace id"
// @Param CreateAppRequest body cluster_user.DevSpaceRequest true "kubeconfig"
// @Success 200 {object} model.ClusterUserModel
// @Router /v1/dev_space/{id} [put]
func Update(c *gin.Context) {
	var req DevSpaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind dev space params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	devSpaceId := cast.ToUint64(c.Param("id"))
	sDec, err := base64.StdEncoding.DecodeString(req.KubeConfig)
	if err != nil {
		log.Warnf("bind dev space params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	cu := model.ClusterUserModel{
		ID:         devSpaceId,
		KubeConfig: string(sDec),
	}
	result, err := service.Svc.ClusterUser().Update(c, &cu)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}
