package model

import (
	validator "github.com/go-playground/validator/v10"
	"time"
)

// ClusterUserModel
type ClusterUserModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	UserId    uint64     `gorm:"column:user_id;not null" json:"user_id"`
	ClusterId uint64     `gorm:"column:cluster_id;not null" json:"cluster_id"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
}

// Validate the fields.
func (u *ClusterUserModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName 表名
func (u *ClusterUserModel) TableName() string {
	return "clusters_users"
}
