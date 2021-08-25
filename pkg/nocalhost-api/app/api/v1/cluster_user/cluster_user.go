/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

type ClusterUserCreateRequest struct {
	ID                 *uint64                   `json:"id"`
	ClusterId          *uint64                   `json:"cluster_id" binding:"required"`
	UserId             *uint64                   `json:"user_id" binding:"required"`
	SpaceName          string                    `json:"space_name"`
	Memory             *uint64                   `json:"memory"`
	Cpu                *uint64                   `json:"cpu"`
	ApplicationId      *uint64                   `json:"application_id"`
	ClusterAdmin       *uint64                   `json:"cluster_admin"`
	NameSpace          string                    `json:"namespace"`
	SpaceResourceLimit *SpaceResourceLimit       `json:"space_resource_limit"`
	BaseDevSpaceId     uint64                    `json:"base_dev_space_id"`
	MeshDevInfo        *setupcluster.MeshDevInfo `json:"mesh_dev_info"`
	IsBaseSpace        bool                      `json:"is_base_space"`
}

type ClusterUserGetRequest struct {
	ClusterUserId *uint64 `form:"cluster_user_id" binding:"required"`
}

type ClusterUserShareRequest struct {
	ClusterUserId *uint64  `json:"cluster_user_id" binding:"required"`
	Cooperators   []uint64 `json:"cooperators"`
	Viewers       []uint64 `json:"viewers"`
}

type ClusterUserUnShareRequest struct {
	ClusterUserId *uint64  `json:"cluster_user_id" binding:"required"`
	Users         []uint64 `json:"users"`
}

type ClusterUserListQuery struct {
	UserId *uint64 `form:"user_id"`
}

type ClusterUserListV2Query struct {
	OwnerUserId            *uint64 `form:"owner_user_id"`
	ClusterId              *uint64 `form:"cluster_id"`
	SpaceName              string  `form:"space_name"`
	IsCanBeUsedAsBaseSpace bool    `form:"is_can_be_used_as_base_space"`
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

func (srl *SpaceResourceLimit) ResourceLimitIsSet() bool {

	return srl != nil && (srl.SpaceReqMem != "" ||
		srl.SpaceReqCpu != "" ||
		srl.SpaceLimitsMem != "" ||
		srl.SpaceLimitsCpu != "" ||
		srl.SpaceLbCount != "" ||
		srl.SpacePvcCount != "" ||
		srl.SpaceStorageCapacity != "" ||
		srl.SpaceEphemeralStorage != "" ||
		srl.ContainerReqMem != "" ||
		srl.ContainerReqCpu != "" ||
		srl.ContainerLimitsMem != "" ||
		srl.ContainerLimitsCpu != "" ||
		srl.ContainerEphemeralStorage != "")
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
