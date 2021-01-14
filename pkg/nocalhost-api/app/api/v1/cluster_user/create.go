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

func ValidSpaceResourceLimit(resLimit SpaceResourceLimit) (bool, string) {
	reg, _ := regexp.Compile("^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$")
	numReg, _ := regexp.Compile("^([+-]?[0-9]+)$")

	var message []string
	if resLimit.SpaceReqMem != "" && !reg.MatchString(resLimit.SpaceReqMem) {
		message = append(message, "space_req_mem")
	}
	if resLimit.SpaceLimitsMem != "" && !reg.MatchString(resLimit.SpaceLimitsMem) {
		message = append(message, "space_limits_mem")
	}
	if resLimit.SpaceReqCpu != "" && !reg.MatchString(resLimit.SpaceReqCpu) {
		message = append(message, "space_req_cpu")
	}
	if resLimit.SpaceLimitsCpu != "" && !reg.MatchString(resLimit.SpaceLimitsCpu) {
		message = append(message, "space_limits_cpu")
	}
	if resLimit.SpaceLbCount > 0 && !numReg.MatchString(strconv.Itoa(resLimit.SpaceLbCount)) {
		message = append(message, "space_lb_count")
	}
	if resLimit.SpacePvcCount > 0 && !numReg.MatchString(strconv.Itoa(resLimit.SpacePvcCount)) {
		message = append(message, "space_pvc_count")
	}
	if resLimit.SpaceStorageCapacity != "" && !reg.MatchString(resLimit.SpaceStorageCapacity) {
		message = append(message, "space_storage_capacity")
	}
	if resLimit.SpaceEphemeralStorage != "" && !reg.MatchString(resLimit.SpaceEphemeralStorage) {
		message = append(message, "space_ephemeral_storage")
	}
	if resLimit.ContainerReqCpu != "" && !reg.MatchString(resLimit.ContainerReqCpu) {
		message = append(message, "container_req_cpu")
	}
	if resLimit.ContainerReqCpu != "" && !reg.MatchString(resLimit.ContainerReqCpu) {
		message = append(message, "container_req_cpu")
	}
	if resLimit.ContainerLimitsMem != "" && !reg.MatchString(resLimit.ContainerLimitsMem) {
		message = append(message, "container_limits_mem")
	}
	if resLimit.ContainerLimitsCpu != "" && !reg.MatchString(resLimit.ContainerLimitsCpu) {
		message = append(message, "container_limits_cpu")
	}
	if resLimit.ContainerEphemeralStorage != "" && !reg.MatchString(resLimit.ContainerEphemeralStorage) {
		message = append(message, "container_ephemeral_storage")
	}
	if len(message) > 0 {
		return false, strings.Join(message, ",")
	}
	return true, strings.Join(message, ",")
}
