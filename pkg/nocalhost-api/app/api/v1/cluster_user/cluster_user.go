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

type ClusterUserCreateRequest struct {
	ID                 *uint64             `json:"id"`
	ClusterId          *uint64             `json:"cluster_id" binding:"required"`
	UserId             *uint64             `json:"user_id" binding:"required"`
	SpaceName          string              `json:"space_name"`
	Memory             *uint64             `json:"memory"`
	Cpu                *uint64             `json:"cpu"`
	ApplicationId      *uint64             `json:"application_id"`
	ClusterAdmin       *uint64             `json:"cluster_admin"`
	NameSpace          string              `json:"namespace"`
	SpaceResourceLimit *SpaceResourceLimit `json:"space_resource_limit"`
}

type ClusterUserListQuery struct {
	UserId *uint64 `form:"user_id"`
}

type SpaceResourceLimit struct {
	SpaceReqMem               string `json:"space_req_mem"`
	SpaceReqCpu               string `json:"space_req_cpu"`
	SpaceLimitsMem            string `json:"space_limits_mem"`
	SpaceLimitsCpu            string `json:"space_limits_cpu"`
	SpaceLbCount              string `json:"space_lb_count"`
	SpacePvcCount             string `json:"space_pvc_count"`
	SpaceStorageCapacity      string `json:"space_storage_capacity"`
	SpaceEphemeralStorage     string `json:"space_ephemeral_storage"`
	ContainerReqMem           string `json:"container_req_mem"`
	ContainerReqCpu           string `json:"container_req_cpu"`
	ContainerLimitsMem        string `json:"container_limits_mem"`
	ContainerLimitsCpu        string `json:"container_limits_cpu"`
	ContainerEphemeralStorage string `json:"container_ephemeral_storage"`
}

func (srl *SpaceResourceLimit) Validate() bool {
	if srl.SpaceReqMem != "" && srl.ContainerReqMem == "" ||
		srl.SpaceLimitsMem != "" && srl.ContainerLimitsMem == "" ||
		srl.SpaceReqCpu != "" && srl.ContainerReqCpu == "" ||
		srl.SpaceLimitsCpu != "" && srl.ContainerLimitsCpu == "" {
		return false
	}
	return true
}
