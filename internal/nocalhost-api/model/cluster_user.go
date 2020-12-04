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

package model

import (
	"time"

	validator "github.com/go-playground/validator/v10"
)

// ClusterUserModel
type ClusterUserModel struct {
	ID            uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	ApplicationId uint64     `gorm:"column:application_id;not null" json:"application_id"`
	UserId        uint64     `gorm:"column:user_id;not null" json:"user_id"`
	ClusterId     uint64     `gorm:"column:cluster_id;not null" json:"cluster_id"`
	KubeConfig    string     `gorm:"column:kubeconfig;not null" json:"kubeconfig"`
	Memory        uint64     `gorm:"column:memory;not null" json:"memory"`
	Cpu           uint64     `gorm:"column:cpu;not null" json:"cpu"`
	Namespace     string     `gorm:"column:namespace;not null" json:"namespace"`
	Status        *uint64    `gorm:"column:status;default:0" json:"status"`
	CreatedAt     time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"-"`
}

type ClusterUserJoinCluster struct {
	ID                     uint64 `gorm:"column:id" json:"id"`
	UserId                 uint64 `gorm:"column:user_id" json:"user_id"`
	ApplicationId          uint64 `gorm:"column:application_id" json:"application_id"`
	ClusterId              uint64 `gorm:"column:cluster_id" json:"cluster_id"`
	Namespace              string `gorm:"column:namespace" json:"namespace"`
	AdminClusterName       string `gorm:"column:admin_cluster_name" json:"admin_cluster_name"`
	AdminClusterKubeConfig string `gorm:"column:admin_cluster_kubeconfig" json:"admin_cluster_kubeconfig"`
}

// Validate the fields.
func (u *ClusterUserModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName 表名
func (u *ClusterUserModel) TableName() string {
	return "clusters_users"
}
