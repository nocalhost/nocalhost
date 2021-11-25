/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const (
	IsolateSpace SpaceType = "IsolateSpace"
	MeshSpace    SpaceType = "MeshSpace"
)

var DevSpaceOwnTypeOwner SpaceOwnType = SpaceOwnType{"Owner", 1000}
var DevSpaceOwnTypeCooperator SpaceOwnType = SpaceOwnType{"Cooperator", 100}
var DevSpaceOwnTypeViewer SpaceOwnType = SpaceOwnType{"Viewer", 10}
var None SpaceOwnType = SpaceOwnType{"None", 1}

type ClusterUserV2 struct {

	// Intrinsic field
	ID                 uint64       `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	UserId             uint64       `gorm:"column:user_id;not null" json:"user_id"`
	ClusterAdmin       *uint64      `gorm:"column:cluster_admin;default:0" json:"cluster_admin"`
	Namespace          string       `gorm:"column:namespace;not null" json:"namespace"`
	SpaceName          string       `gorm:"column:space_name;not null;type:VARCHAR(100);comment:'default is application[username]'" json:"space_name"`
	ClusterId          uint64       `gorm:"column:cluster_id;not null" json:"cluster_id"`
	IsBaseSpace        bool         `gorm:"column:is_base_space;default:false" json:"is_base_space"`
	BaseDevSpaceId     uint64       `gorm:"column:base_dev_space_id;default:0" json:"base_dev_space_id"`
	TraceHeader        Header       `gorm:"column:trace_header;type:VARCHAR(256);" json:"trace_header"`
	SpaceResourceLimit string       `gorm:"column:space_resource_limit;type:VARCHAR(1024);" json:"space_resource_limit"`
	CreatedAt          time.Time    `gorm:"column:created_at" json:"created_at"`
	SleepAt            *time.Time   `gorm:"column:sleep_at;type:timestamp" json:"sleep_at"`
	IsAsleep           bool         `gorm:"column:is_asleep;not null;default:false" json:"is_asleep"`
	SleepConfig        *SleepConfig `gorm:"column:sleep_config;type:VARCHAR(1024);" json:"sleep_config"`

	// ext field
	*ClusterUserExt
}

type SpaceType string

type SpaceOwnType struct {
	Str      string
	Priority int
}

type ClusterUserExt struct {
	ClusterName           string        `json:"cluster_name"`
	SpaceType             SpaceType     `json:"space_type"`
	SpaceOwnType          SpaceOwnType  `json:"space_own_type"`
	ResourceLimitSet      bool          `json:"resource_limit_set"`
	CooperUser            []*UserSimple `json:"cooper_user"`
	ViewerUser            []*UserSimple `json:"viewer_user"`
	Owner                 *UserSimple   `json:"owner"`
	Modifiable            bool          `json:"modifiable"`
	Deletable             bool          `json:"deletable"`
	BaseDevSpaceName      string        `json:"base_dev_space_name"`
	BaseDevSpaceNameSpace string        `json:"base_dev_space_namespace"`
}

// ClusterUserModel
type ClusterUserModel struct {
	ID uint64 `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`

	// Deprecated
	ApplicationId      uint64       `gorm:"column:application_id;not null" json:"application_id"`
	UserId             uint64       `gorm:"column:user_id;not null" json:"user_id"`
	SpaceName          string       `gorm:"column:space_name;not null;type:VARCHAR(100);comment:'default is application[username]'" json:"space_name"`
	ClusterId          uint64       `gorm:"column:cluster_id;not null" json:"cluster_id"`
	KubeConfig         string       `gorm:"column:kubeconfig;not null" json:"kubeconfig"`
	Memory             uint64       `gorm:"column:memory;not null" json:"memory"`
	Cpu                uint64       `gorm:"column:cpu;not null" json:"cpu"`
	SpaceResourceLimit string       `gorm:"column:space_resource_limit;type:VARCHAR(1024);" json:"space_resource_limit"`
	Namespace          string       `gorm:"column:namespace;not null" json:"namespace"`
	Status             *uint64      `gorm:"column:status;default:0" json:"status"`
	ClusterAdmin       *uint64      `gorm:"column:cluster_admin;default:0" json:"cluster_admin"`
	IsBaseSpace        bool         `gorm:"column:is_base_space;default:false" json:"is_base_space"`
	BaseDevSpaceId     uint64       `gorm:"column:base_dev_space_id;default:0" json:"base_dev_space_id"`
	TraceHeader        Header       `gorm:"cloumn:trace_header;type:VARCHAR(256);" json:"trace_header"`
	CreatedAt          time.Time    `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time    `gorm:"column:updated_at" json:"-"`
	DeletedAt          *time.Time   `gorm:"column:deleted_at" json:"-"`
	SleepAt            *time.Time   `gorm:"column:sleep_at;type:timestamp" json:"sleep_at"`
	IsAsleep           bool         `gorm:"column:is_asleep;not null;default:false" json:"is_asleep"`
	SleepConfig        *SleepConfig `gorm:"column:sleep_config;type:VARCHAR(1024);" json:"sleep_config"`
}

func (cu *ClusterUserModel) IsClusterAdmin() bool {
	return cu != nil && cu.ClusterAdmin != nil && *cu.ClusterAdmin != uint64(0)
}

func (cu *ClusterUserV2) IsClusterAdmin() bool {
	return cu != nil && cu.ClusterAdmin != nil && *cu.ClusterAdmin != uint64(0)
}

type ClusterUserPluginModel struct {
	ID uint64 `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`

	// Deprecated
	ApplicationId      uint64     `gorm:"column:application_id;not null" json:"application_id"`
	UserId             uint64     `gorm:"column:user_id;not null" json:"user_id"`
	SpaceName          string     `gorm:"column:space_name;not null;type:VARCHAR(100);comment:'default is application[username]'" json:"space_name"`
	ClusterId          uint64     `gorm:"column:cluster_id;not null" json:"cluster_id"`
	KubeConfig         string     `gorm:"column:kubeconfig;not null" json:"kubeconfig"`
	Memory             uint64     `gorm:"column:memory;not null" json:"memory"`
	Cpu                uint64     `gorm:"column:cpu;not null" json:"cpu"`
	SpaceResourceLimit string     `gorm:"cloumn:space_resource_limit;type:VARCHAR(1024);" json:"space_resource_limit"`
	Namespace          string     `gorm:"column:namespace;not null" json:"namespace"`
	Status             *uint64    `gorm:"column:status;default:0" json:"status"`
	CreatedAt          time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt          *time.Time `gorm:"column:deleted_at" json:"-"`

	DevStartAppendCommand []string `json:"dev_start_append_command"`
	// from clusters
	StorageClass string `json:"storage_class" gorm:"column:storage_class"`
}

type ClusterUserJoinCluster struct {
	ID                     uint64 `gorm:"column:id" json:"id"`
	UserId                 uint64 `gorm:"column:user_id" json:"user_id"`
	ApplicationId          uint64 `gorm:"column:application_id" json:"application_id"`
	ClusterId              uint64 `gorm:"column:cluster_id" json:"cluster_id"`
	Namespace              string `gorm:"column:namespace" json:"namespace"`
	AdminClusterName       string `gorm:"column:admin_cluster_name" json:"admin_cluster_name"`
	AdminClusterKubeConfig string `gorm:"column:admin_cluster_kubeconfig" json:"admin_cluster_kubeconfig"`
}

type ClusterUserJoinClusterAndAppAndUser struct {
	ID                 uint64    `gorm:"column:id" json:"id"`
	UserId             uint64    `gorm:"column:user_id" json:"user_id"`
	UserName           string    `gorm:"column:user_name" json:"user_name"`
	SpaceName          string    `gorm:"column:space_name" json:"space_name"`
	ClusterAdmin       *uint64   `gorm:"column:cluster_admin;default:0" json:"cluster_admin"`
	ClusterId          uint64    `gorm:"column:cluster_id" json:"cluster_id"`
	ClusterName        string    `gorm:"column:cluster_name" json:"cluster_name"`
	KubeConfig         string    `gorm:"column:kubeconfig" json:"kubeconfig"`
	SpaceResourceLimit string    `gorm:"cloumn:space_resource_limit" json:"space_resource_limit"`
	Namespace          string    `gorm:"column:namespace" json:"namespace"`
	Status             *uint64   `gorm:"column:status" json:"status"`
	CreatedAt          time.Time `gorm:"column:created_at" json:"created_at"`
}

// Validate the fields.
func (u *ClusterUserModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName
func (u *ClusterUserModel) TableName() string {
	return "clusters_users"
}

type Header struct {
	TraceKey   string `json:"key"`
	TraceValue string `json:"value"`
	TraceType  string `json:"type"`
}

func (h *Header) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	b, ok := value.([]byte)
	if !ok {
		return errors.Errorf("value is not []byte, value: %v", value)
	}
	return json.Unmarshal(b, h)
}

func (h Header) Value() (driver.Value, error) {
	return json.Marshal(h)
}
