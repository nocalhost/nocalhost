package cluster_user

type ClusterUserCreateRequest struct {
	ID            *uint64 `json:"id"`
	ClusterId     *uint64 `json:"cluster_id" binding:"required"`
	UserId        *uint64 `json:"user_id" binding:"required"`
	SpaceName     string  `json:"space_name"`
	Memory        *uint64 `json:"memory"`
	Cpu           *uint64 `json:"cpu"`
	ApplicationId *uint64 `json:"application_id"`
	NameSpace     string  `json:"namespace"`
}
