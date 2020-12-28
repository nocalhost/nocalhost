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

// ApplicationModel
type ApplicationModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Context   string     `json:"context" gorm:"column:context;not null" binding:"required"`
	UserId    uint64     `gorm:"column:user_id;not null" json:"user_id"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
	Status    uint8      `json:"status" gorm:"column:status;not null" binding:"required"`
}

type PluginApplicationModel struct {
	ID            uint64 `gorm:"column:id" json:"id"`
	Context       string `json:"context" gorm:"column:context"`
	UserId        uint64 `gorm:"column:user_id" json:"-"`
	Status        uint64 `json:"status" gorm:"column:status"`
	ClusterId     uint64 `json:"cluster_id" gorm:"column:cluster_id"`
	SpaceName     string `json:"space_name" gorm:"column:space_name"`
	KubeConfig    string `json:"kubeconfig" gorm:"column:kubeconfig"`
	StorageClass  string `json:"storage_class" gorm:"column:storage_class"`
	Memory        uint64 `json:"memory" gorm:"column:memory"`
	Cpu           uint64 `json:"cpu" gorm:"column:cpu"`
	NameSpace     string `json:"namespace" gorm:"column:namespace"`
	InstallStatus uint64 `json:"install_status" gorm:"column:install_status"`
	DevSpaceId    uint64 `json:"devspace_id" gorm:"column:devspace_id"`
}

// Validate the fields.
func (u *ApplicationModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName
func (u *ApplicationModel) TableName() string {
	return "applications"
}
