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

package cluster

import "time"

type CreateClusterRequest struct {
	Name         string `json:"name" binding:"required"`
	KubeConfig   string `json:"kubeconfig" binding:"required" example:"base64encode(value)"`
	StorageClass string `json:"storage_class"`
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

type StorageClassRequest struct {
	KubeConfig string `json:"kubeconfig" example:"base64encode(value)"`
}

type StorageClassResponse struct {
	TypeName []string `json:"type_name"`
}

type UpdateClusterRequest struct {
	StorageClass string `json:"storage_class"`
}

type ClusterDetailResponse struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Info         string    `json:"info"`
	UserId       uint64    `json:"user_id"`
	Server       string    `json:"server"`
	KubeConfig   string    `json:"kubeconfig"`
	StorageClass string    `json:"storage_class"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
}
