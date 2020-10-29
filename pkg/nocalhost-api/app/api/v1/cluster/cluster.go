package cluster

// 创建集群请求
type CreateClusterRequest struct {
	Name string `json:"name" binding:"required"`
	Marks string `json:"marks"`
	KubeConfig string `json:"kubeconfig" binding:"required"`
}

