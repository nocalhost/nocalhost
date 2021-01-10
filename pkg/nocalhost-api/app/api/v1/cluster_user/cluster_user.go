package cluster_user

type ClusterUserCreateRequest struct {
	ClusterId          *uint64             `json:"cluster_id" binding:"required"`
	UserId             *uint64             `json:"user_id" binding:"required"`
	SpaceName          string              `json:"space_name"`
	Memory             *uint64             `json:"memory"`
	Cpu                *uint64             `json:"cpu"`
	SpaceResourceLimit *SpaceResourceLimit `json:"space_resource_limit"`
}

type SpaceResourceLimit struct {
	SpaceReqMem          string `json:"space_req_mem"`
	SpaceReqCpu          string `json:"space_req_cpu"`
	SpaceLimitsMem       string `json:"space_limits_mem"`
	SpaceLimitsCpu       string `json:"space_limits_cpu"`
	SpacePvcCount        int    `json:"space_pvc_count"`
	SpaceStorageCapacity string `json:"space_storage_capacity"`
	SpaceLbCount         int    `json:"space_lb_count"`

	ContainerReqMem    string `json:"container_req_mem"`
	ContainerReqCpu    string `json:"container_req_cpu"`
	ContainerLimitsMem string `json:"container_limits_mem"`
	ContainerLimitsCpu string `json:"container_limits_cpu"`
}
