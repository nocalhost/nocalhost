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

// ApplicationModel
type ApplicationModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Context   string     `gorm:"column:context;not null"              json:"context"    binding:"required"`
	UserId    uint64     `gorm:"column:user_id;not null"              json:"user_id"`
	CreatedAt time.Time  `gorm:"column:created_at"                    json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"                    json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at"                    json:"-"`
	Public    uint8      `gorm:"column:public;not null"               json:"public"     binding:"required"`
	Status    uint8      `gorm:"column:status;not null"               json:"status"     binding:"required"`
	Editable  uint8      `                                            json:"editable"`
}

type PluginApplicationModel struct {
	ID                    uint64 `gorm:"column:id"             json:"id"`
	Context               string `gorm:"column:context"        json:"context"`
	UserId                uint64 `gorm:"column:user_id"        json:"-"`
	Status                uint64 `gorm:"column:status"         json:"status"`
	Public                uint8  `gorm:"column:public"         json:"public"`
	ClusterId             uint64 `gorm:"column:cluster_id"     json:"cluster_id"`
	SpaceName             string `gorm:"column:space_name"     json:"space_name"`
	KubeConfig            string `gorm:"column:kubeconfig"     json:"kubeconfig"`
	StorageClass          string `gorm:"column:storage_class"  json:"storage_class"`
	Memory                uint64 `gorm:"column:memory"         json:"memory"`
	Cpu                   uint64 `gorm:"column:cpu"            json:"cpu"`
	NameSpace             string `gorm:"column:namespace"      json:"namespace"`
	InstallStatus         uint64 `gorm:"column:install_status" json:"install_status"`
	DevSpaceId            uint64 `gorm:"column:devspace_id"    json:"devspace_id"`
	DevStartAppendCommand string `                             json:"dev_start_append_command"`
}

func (u *ApplicationModel) FillEditable(admin bool, currentUser uint64) {
	if admin || currentUser == u.UserId {
		u.Editable = 1
	} else {
		u.Editable = 0
	}
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
