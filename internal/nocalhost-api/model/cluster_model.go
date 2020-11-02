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

// ClusterModel
type ClusterModel struct {
	ID         uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Name       string     `json:"name" gorm:"column:name;not null" binding:"required" validate:"min=1,max=32"`
	Marks      string     `json:"marks" gorm:"column:marks;not null" json:"marks"`
	UserId     uint64     `gorm:"column:user_id;not null" json:"-"`
	KubeConfig string     `json:"kubeconfig" gorm:"column:kubeconfig;not null" binding:"required"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt  time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt  *time.Time `gorm:"column:deleted_at" json:"-"`
}

// Validate the fields.
func (u *ClusterModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName 表名
func (u *ClusterModel) TableName() string {
	return "clusters"
}
