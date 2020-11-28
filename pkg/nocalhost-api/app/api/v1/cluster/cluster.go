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

package cluster

// 创建集群请求
type CreateClusterRequest struct {
	Name       string `json:"name" binding:"required"`
	KubeConfig string `json:"kubeconfig" binding:"required" example:"base64encode(value)"`
}

type KubeConfig struct {
	Clusters []Clusters
}

type Clusters struct {
	Cluster Cluster
	Name    string
}

type Cluster struct {
	CertificateAuthorityData string
	Server                   string
}
