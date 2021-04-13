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

// PrePullModel
type PrePullModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Images    string     `gorm:"column:images;not null"               json:"images"`
	DeletedAt *time.Time `gorm:"column:deleted_at"                    json:"-"`
}

// Validate the fields.
func (u *PrePullModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName
func (u *PrePullModel) TableName() string {
	return "pre_pull"
}
