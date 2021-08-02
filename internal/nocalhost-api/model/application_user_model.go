/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
