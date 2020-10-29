package model

import (
	validator "github.com/go-playground/validator/v10"
	"time"
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

// Validate the fields.
func (u *ApplicationClusterModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName 表名
func (u *ApplicationClusterModel) TableName() string {
	return "applications_clusters"
}
