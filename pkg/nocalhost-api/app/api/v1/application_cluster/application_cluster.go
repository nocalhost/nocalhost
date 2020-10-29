package application_cluster

// 关联集群请求体
type ApplicationClusterRequest struct {
	ClusterId *uint64 `json:"cluster_id" binding:"required"`
}
