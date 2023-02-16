/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import "time"

type CreateClusterRequest struct {
	Name           string `json:"name" binding:"required"`
	KubeConfig     string `json:"kubeconfig" binding:"required" example:"base64encode(value)"`
	StorageClass   string `json:"storage_class"`
	ExtraApiServer string `json:"extra_api_server" binding:"omitempty,url"`
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

type Namespace struct {
	Namespace string `json:"namespace"`
}

type ClusterUserMigrateRequest struct {
	Migrate []NsAndUsers
}

type NsAndUsers struct {
	Namespace string
	Users     []string
}
