/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

import (
	"time"

	validator "github.com/go-playground/validator/v10"
)

// ApplicationModel
type ApplicationModel struct {
	ID              uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Context         string     `json:"context" gorm:"column:context;not null" binding:"required"`
	UserId          uint64     `gorm:"column:user_id;not null" json:"user_id"`
	CreatedAt       time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt       *time.Time `gorm:"column:deleted_at" json:"-"`
	UserName        string     `json:"user_name"`
	Public          uint8      `json:"public" gorm:"column:public;not null" binding:"required"`
	Status          uint8      `json:"status" gorm:"column:status;not null" binding:"required"`
	Editable        uint8      `json:"editable"`
	ApplicationType string     `json:"application_type"`
}

type PluginApplicationModel struct {
	ID                    uint64 `gorm:"column:id" json:"id"`
	Context               string `json:"context" gorm:"column:context"`
	UserId                uint64 `gorm:"column:user_id" json:"-"`
	Status                uint64 `json:"status" gorm:"column:status"`
	Public                uint8  `json:"public" gorm:"column:public"`
	ClusterId             uint64 `json:"cluster_id" gorm:"column:cluster_id"`
	SpaceName             string `json:"space_name" gorm:"column:space_name"`
	KubeConfig            string `json:"kubeconfig" gorm:"column:kubeconfig"`
	StorageClass          string `json:"storage_class" gorm:"column:storage_class"`
	Memory                uint64 `json:"memory" gorm:"column:memory"`
	Cpu                   uint64 `json:"cpu" gorm:"column:cpu"`
	NameSpace             string `json:"namespace" gorm:"column:namespace"`
	InstallStatus         uint64 `json:"install_status" gorm:"column:install_status"`
	DevSpaceId            uint64 `json:"devspace_id" gorm:"column:devspace_id"`
	DevStartAppendCommand string `json:"dev_start_append_command"`
}

func (u *ApplicationModel) FillEditable(admin bool, currentUser uint64) {
	if admin || currentUser == u.UserId {
		u.Editable = 1
	} else {
		u.Editable = 0
	}
}

func (u *ApplicationModel) FillUserName(usrName string){
	u.UserName = usrName
}

func (u *ApplicationModel) FillApplicationType(ApplicationType string) {
	u.ApplicationType = ApplicationType
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
