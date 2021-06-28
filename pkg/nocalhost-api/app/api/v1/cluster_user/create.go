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
// @Tags DevSpace
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body cluster_user.ClusterUserCreateRequest true "cluster user info"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/{id} [post]
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
			log.Errorf("Initial devSpace fail. Incorrect Resource limit parameter [ %v ] format.", message)
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

	var message msgList = []string{}
	message.appendWhileMatch(resLimit.SpaceReqMem, "space_req_mem", regMem)
	message.appendWhileMatch(resLimit.SpaceLimitsMem, "space_limits_mem", regMem)
	message.appendWhileMatch(resLimit.SpaceReqCpu, "space_req_cpu", regCpu)
	message.appendWhileMatch(resLimit.SpaceLimitsCpu, "space_limits_cpu", regCpu)
	message.appendWhileMatch(resLimit.SpaceLbCount, "space_lb_count", numReg)
	message.appendWhileMatch(resLimit.SpacePvcCount, "space_pvc_count", numReg)
	message.appendWhileMatch(resLimit.SpaceStorageCapacity, "space_storage_capacity", regStorage)
	message.appendWhileMatch(resLimit.ContainerReqMem, "container_req_mem", regMem)
	message.appendWhileMatch(resLimit.ContainerReqCpu, "container_req_cpu", regCpu)
	message.appendWhileMatch(resLimit.ContainerLimitsMem, "container_limits_mem", regMem)
	message.appendWhileMatch(resLimit.ContainerLimitsCpu, "container_limits_cpu", regCpu)

	if len(message) > 0 {
		return false, strings.Join(message, ",")
	}
	return true, strings.Join(message, ",")
}

type msgList []string

func (l *msgList) appendWhileMatch(value string, key string, reg *regexp.Regexp) {
	if len(value) > 0 && !reg.MatchString(value) {
		*l = append(*l, key)
	}
}
