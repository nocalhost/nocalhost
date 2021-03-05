package model

import "time"
import validator "github.com/go-playground/validator/v10"

// ApplicationModel
type ApplicationUserModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	UserId    uint64     `gorm:"column:user_id;not null" json:"user_id"`
	ClusterId uint64     `gorm:"column:cluster_id;not null" json:"cluster_id"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
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
