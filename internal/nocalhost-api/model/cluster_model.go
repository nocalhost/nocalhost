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

package model

import (
	"time"

	validator "github.com/go-playground/validator/v10"
)

// ClusterModel
type ClusterModel struct {
	ID           uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id"                                                            json:"id"`
	Name         string     `gorm:"column:name;not null"                                                                            json:"name"          binding:"required" validate:"min=1,max=32"`
	Info         string     `gorm:"column:info;"                                                                                    json:"info"`
	UserId       uint64     `gorm:"column:user_id;not null"                                                                         json:"user_id"`
	Server       string     `gorm:"column:server;not null"                                                                          json:"server"`
	KubeConfig   string     `gorm:"column:kubeconfig;not null"                                                                      json:"kubeconfig"    binding:"required"`
	StorageClass string     `gorm:"column:storage_class;not null;type:VARCHAR(100);comment:'empty means use default storage class'" json:"storage_class"`
	CreatedAt    time.Time  `gorm:"column:created_at"                                                                               json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"                                                                               json:"-"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"                                                                               json:"-"`
}

type ClusterDetailModel struct {
	ID              uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Name            string     `gorm:"column:name;not null"                 json:"name"              binding:"required" validate:"min=1,max=32"`
	Info            string     `gorm:"column:info;"                         json:"info"`
	UserId          uint64     `gorm:"column:user_id;not null"              json:"user_id"`
	Server          string     `gorm:"column:server;not null"               json:"server"`
	KubeConfig      string     `gorm:"column:kubeconfig;not null"           json:"kubeconfig"        binding:"required"`
	StorageClass    string     `gorm:"column:storage_class;not null"        json:"storage_class"`
	CreatedAt       time.Time  `gorm:"column:created_at"                    json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"                    json:"-"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"                    json:"-"`
	IsReady         bool       `                                            json:"is_ready"`
	NotReadyMessage string     `                                            json:"not_ready_message"`
}

type ClusterList struct {
	ID              uint64    `gorm:"column:id"                     json:"id"`
	ClusterName     string    `gorm:"column:name"                   json:"name"`
	UsersCount      uint64    `gorm:"column:users_count"            json:"users_count"`
	KubeConfig      string    `gorm:"column:kubeconfig"             json:"-"`
	StorageClass    string    `gorm:"column:storage_class;not null" json:"storage_class"`
	Info            string    `gorm:"column:info"                   json:"info"`
	UserId          uint64    `gorm:"column:user_id;not null"       json:"user_id"`
	CreatedAt       time.Time `gorm:"column:created_at"             json:"created_at"`
	IsReady         bool      `                                     json:"is_ready"`
	NotReadyMessage string    `                                     json:"not_ready_message"`
	HasDevSpace     bool      `                                     json:"has_dev_space"`
}

// Validate the fields.
func (u *ClusterModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName
func (u *ClusterModel) TableName() string {
	return "clusters"
}
