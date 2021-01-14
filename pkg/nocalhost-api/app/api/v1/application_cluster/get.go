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

package application_cluster

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

// GetBound Get the list of clusters associated with the application
// @Summary Get the list of clusters associated with the application（Abandoned）
// @Description Get the list of clusters associated with the application（Abandoned）
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} model.ApplicationClusterModel
// @Router /v1/application/{id}/bound_cluster [get]
func GetBound(c *gin.Context) {
	applicationId := cast.ToUint64(c.Param("id"))

	// application permission
	if _, err := service.Svc.ApplicationSvc().Get(c, applicationId); err != nil {
		api.SendResponse(c, errno.ErrPermissionApplication, nil)
		return
	}

	// get bound list
	result, err := service.Svc.ApplicationClusterSvc().GetJoinCluster(c, applicationId)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationBoundClusterList, nil)
		return
	}
	api.SendResponse(c, errno.OK, result)
	return
}
