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
