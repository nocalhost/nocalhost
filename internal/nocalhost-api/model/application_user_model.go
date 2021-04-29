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

// ApplicationClusterModel
type ApplicationUserModel struct {
	ID            uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	ApplicationId uint64     `gorm:"column:application_id;UNIQUE_INDEX:uidx_userapp;not null" json:"application_id"`
	UserId        uint64     `gorm:"column:user_id;UNIQUE_INDEX:uidx_userapp;not null" json:"user_id"`
	CreatedAt     time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt     *time.Time `gorm:"column:deleted_at" json:"-"`
}

// Validate the fields.
func (u *ApplicationUserModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName
func (u *ApplicationUserModel) TableName() string {
	return "applications_users"
}
