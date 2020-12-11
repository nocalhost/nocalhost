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

// ApplicationClusterModel
type ApplicationClusterModel struct {
	ID            uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	ApplicationId uint64     `gorm:"column:application_id;not null" json:"application_id"`
	ClusterId     uint64     `gorm:"column:cluster_id;not null" json:"cluster_id"`
	CreatedAt     time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"-"`
}

// ApplicationCluserJoinClusterModel
type ApplicationClusterJoinModel struct {
	ApplicationId               uint64    `gorm:"column:application_id" json:"application_id"`
	ClusterId                   uint64    `gorm:"column:cluster_id" json:"cluster_id"`
	ClusterName                 string    `gorm:"column:cluster_name" json:"cluster_name"`
	ClusterDevSpaceCount        uint64    `gorm:"column:dev_space_count" json:"dev_space_count"`
	ClusterInfo                 string    `gorm:"column:cluster_info" json:"cluster_info"`
	ClusterStatus               uint64    `gorm:"column:cluster_status" json:"cluster_status"`
	ApplicationClusterCreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

// Validate the fields.
func (u *ApplicationClusterModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName
func (u *ApplicationClusterModel) TableName() string {
	return "applications_clusters"
}
