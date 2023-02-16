/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

import (
	v1 "k8s.io/api/core/v1"
	"time"

	validator "github.com/go-playground/validator/v10"
)

type ClusterPack interface {
	GetClusterId() uint64
	GetKubeConfig() string
	GetClusterServer() string
	GetClusterName() string
	GetExtraApiServer() string
}

func (cl *ClusterList) GetClusterId() uint64 {
	return cl.ID
}

func (cl *ClusterList) GetKubeConfig() string {
	return cl.KubeConfig
}

func (cl *ClusterList) GetClusterServer() string {
	return cl.Server
}

func (cl *ClusterList) GetClusterName() string {
	return cl.ClusterName
}

func (cl *ClusterList) GetExtraApiServer() string {
	return cl.ExtraApiServer
}

func (cm *ClusterModel) GetClusterId() uint64 {
	return cm.ID
}

func (cm *ClusterModel) GetKubeConfig() string {
	return cm.KubeConfig
}

func (cm *ClusterModel) GetClusterServer() string {
	return cm.Server
}

func (cm *ClusterModel) GetClusterName() string {
	return cm.Name
}

func (cm *ClusterModel) GetExtraApiServer() string {
	return cm.ExtraApiServer
}

// ClusterModel
type ClusterModel struct {
	ID             uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Name           string     `json:"name" gorm:"column:name;not null" binding:"required" validate:"min=1,max=32"`
	Info           string     `json:"info" gorm:"column:info;"`
	UserId         uint64     `gorm:"column:user_id;not null" json:"user_id"`
	Server         string     `gorm:"column:server;not null" json:"server"`
	ExtraApiServer string     `gorm:"column:extra_api_server" json:"extra_api_server"`
	KubeConfig     string     `json:"kubeconfig" gorm:"column:kubeconfig;not null" binding:"required"`
	StorageClass   string     `json:"storage_class" gorm:"column:storage_class;not null;type:VARCHAR(100);comment:'empty means use default storage class'"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt      *time.Time `gorm:"column:deleted_at" json:"-"`
}

type ClusterList struct {
	ID              uint64    `gorm:"column:id" json:"id"`
	ClusterName     string    `gorm:"column:name" json:"name"`
	UsersCount      uint64    `gorm:"column:users_count" json:"users_count"`
	KubeConfig      string    `gorm:"column:kubeconfig" json:"-"`
	StorageClass    string    `json:"storage_class" gorm:"column:storage_class;not null"`
	Info            string    `gorm:"column:info" json:"info"`
	UserId          uint64    `gorm:"column:user_id;not null" json:"user_id"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
	IsReady         bool      `json:"is_ready"`
	NotReadyMessage string    `json:"not_ready_message"`
	HasDevSpace     bool      `json:"has_dev_space"`
	Server          string    `gorm:"column:server;not null" json:"server"`
	ExtraApiServer  string    `gorm:"column:extra_api_server" json:"extra_api_server"`
	Modifiable      bool      `json:"modifiable"`
}

type ClusterListVo struct {
	ClusterList
	Resources []Resource `json:"resources"`

	// the user create the cluster
	UserName string `json:"user_name"`
}

type Resource struct {
	ResourceName v1.ResourceName `json:"resource_name"`
	Capacity     float64         `json:"capacity"`
	Used         float64         `json:"used"`
	Percentage   float64         `json:"percentage"`
}

func (receiver Resource) Equals(resource Resource) bool {
	return receiver.ResourceName == resource.ResourceName &&
		receiver.Capacity == resource.Capacity &&
		receiver.Used == resource.Used &&
		receiver.Percentage == resource.Percentage
}

// Validate the fields.
func (cm *ClusterModel) Validate() error {
	validate := validator.New()
	return validate.Struct(cm)
}

// TableName
func (cm *ClusterModel) TableName() string {
	return "clusters"
}
