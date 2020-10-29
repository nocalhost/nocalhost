package model

import (
	validator "github.com/go-playground/validator/v10"
	"time"
)

// ApplicationModel
type ApplicationModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Context   string     `json:"context" gorm:"column:context;not null" binding:"required"`
	UserId    uint64     `gorm:"column:user_id;not null" json:"-"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
	Status    uint8      `json:"status" gorm:"column:status;not null" binding:"required"`
}

// Validate the fields.
func (u *ApplicationModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// TableName 表名
func (u *ApplicationModel) TableName() string {
	return "applications"
}
