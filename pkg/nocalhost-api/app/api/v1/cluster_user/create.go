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
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"regexp"
	"strings"
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
	// Validate DevSpace Resource limit parameter format.
	if req.SpaceResourceLimit != nil {
		flag, message := ValidSpaceResourceLimit(*req.SpaceResourceLimit)
		if !flag {
			log.Errorf("Create devSpace fail. Incorrect Resource limit parameter [ %v ] format.", message)
			api.SendResponse(c, errno.ErrFormatResourceLimitParam, message)
			return
		}

		if !req.SpaceResourceLimit.Validate() {
			api.SendResponse(c, errno.ErrValidateResourceQuota, nil)
			return
		}
	}
	applicationId := uint64(0)
	req.ApplicationId = &applicationId
	devSpace := NewDevSpace(req, c, []byte{})
	result, err := devSpace.Create()
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

func ValidSpaceResourceLimit(resLimit SpaceResourceLimit) (bool, string) {
	regMem, _ := regexp.Compile("^([+-]?[0-9.]+)Mi$")
	regCpu, _ := regexp.Compile("^([+-]?[0-9.]+)$")
	regStorage, _ := regexp.Compile("^([+-]?[0-9.]+)Gi$")
	numReg, _ := regexp.Compile("^([+-]?[0-9]+)$")

	var message []string
	if len(resLimit.SpaceReqMem) > 0 && !regMem.MatchString(resLimit.SpaceReqMem) {
		message = append(message, "space_req_mem")
	}
	if len(resLimit.SpaceLimitsMem) > 0 && !regMem.MatchString(resLimit.SpaceLimitsMem) {
		message = append(message, "space_limits_mem")
	}
	if len(resLimit.SpaceReqCpu) > 0 && !regCpu.MatchString(resLimit.SpaceReqCpu) {
		message = append(message, "space_req_cpu")
	}
	if len(resLimit.SpaceLimitsCpu) > 0 && !regCpu.MatchString(resLimit.SpaceLimitsCpu) {
		message = append(message, "space_limits_cpu")
	}
	if len(resLimit.SpaceLbCount) > 0 && !numReg.MatchString(resLimit.SpaceLbCount) {
		message = append(message, "space_lb_count")
	}
	if len(resLimit.SpacePvcCount) > 0 && !numReg.MatchString(resLimit.SpacePvcCount) {
		message = append(message, "space_pvc_count")
	}
	if len(resLimit.SpaceStorageCapacity) > 0 && !regStorage.MatchString(resLimit.SpaceStorageCapacity) {
		message = append(message, "space_storage_capacity")
	}
	if len(resLimit.SpaceEphemeralStorage) > 0 && !regStorage.MatchString(resLimit.SpaceEphemeralStorage) {
		message = append(message, "space_ephemeral_storage")
	}
	if len(resLimit.ContainerReqMem) > 0 && !regMem.MatchString(resLimit.ContainerReqMem) {
		message = append(message, "container_req_mem")
	}
	if len(resLimit.ContainerReqCpu) > 0 && !regCpu.MatchString(resLimit.ContainerReqCpu) {
		message = append(message, "container_req_cpu")
	}
	if len(resLimit.ContainerLimitsMem) > 0 && !regMem.MatchString(resLimit.ContainerLimitsMem) {
		message = append(message, "container_limits_mem")
	}
	if len(resLimit.ContainerLimitsCpu) > 0 && !regCpu.MatchString(resLimit.ContainerLimitsCpu) {
		message = append(message, "container_limits_cpu")
	}
	if len(resLimit.ContainerEphemeralStorage) > 0 && !regStorage.MatchString(resLimit.ContainerEphemeralStorage) {
		message = append(message, "container_ephemeral_storage")
	}
	if len(message) > 0 {
		return false, strings.Join(message, ",")
	}
	return true, strings.Join(message, ",")
}
