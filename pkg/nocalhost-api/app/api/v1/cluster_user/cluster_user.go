package cluster_user

type ClusterUserCreateRequest struct {
	ClusterId *uint64 `json:"cluster_id" binding:"required"`
	UserId    *uint64 `json:"user_id" binding:"required"`
	Memory    *uint64 `json:"memory"`
	Cpu       *uint64 `json:"cpu"`
}
