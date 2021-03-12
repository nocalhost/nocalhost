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

package applications

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"k8s.io/apimachinery/pkg/util/validation"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/napp"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
)

// Create Create Application
// @Summary Create Application
// @Description Create Application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body applications.CreateAppRequest true "The application info"
// @Success 200 {object} model.ApplicationModel
// @Router /v1/application [post]
func Create(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("createApplication bind params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	// check application exist
	var applicationContext ApplicationJsonContext
	err := json.Unmarshal([]byte(req.Context), &applicationContext)
	if err != nil {
		api.SendResponse(c, &errno.Errno{Code: 40110, Message: err.Error()}, nil)
		return
	}
	// check required field
	sErr := napp.App.Validate.Struct(&applicationContext)
	if sErr != nil {
		errs := sErr.(validator.ValidationErrors)
		for _, err := range errs {
			api.SendResponse(c, &errno.Errno{Code: 40110, Message: err.StructNamespace() + " is illegal, require: " + err.Param()}, nil)
			return
		}
	}
	// check application name is match DNS-1123
	errs := validation.IsDNS1123Label(applicationContext.ApplicationName)
	if len(errs) > 0 {
		api.SendResponse(c, &errno.Errno{Code: 40110, Message: errs[0]}, nil)
		return
	}
	existApplication, _ := service.Svc.ApplicationSvc().GetByName(c, applicationContext.ApplicationName)
	if existApplication.ID != 0 {
		api.SendResponse(c, errno.ErrApplicationNameExist, nil)
		return
	}

	// adapt earlier version
	if req.Public==nil {
		u := uint8(1)
		req.Public = &u
	}

	// normal user can't not create public applications
	if !ginbase.IsAdmin(c) {
		deny := uint8(0)
		req.Public = &deny
	}

	userId, _ := c.Get("userId")
	a, err := service.Svc.ApplicationSvc().Create(c, req.Context, *req.Status, *req.Public, userId.(uint64))
	if err != nil {
		log.Warnf("create Application err: %v", err)
		api.SendResponse(c, errno.ErrApplicationCreate, nil)
		return
	}

	api.SendResponse(c, nil, a)
}
