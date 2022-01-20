/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package model

import (
	"time"

	validator "github.com/go-playground/validator/v10"
)

// PrePullModel
type PrePullModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Images    string     `gorm:"column:images;not null" json:"images"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
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
