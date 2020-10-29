package model

import (
	"sync"
	"time"

	"nocalhost/pkg/nocalhost-api/pkg/auth"

	validator "github.com/go-playground/validator/v10"
)

// UserBaseModel
type UserBaseModel struct {
	ID        uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Uuid      string     `gorm:"column:uuid;not null" json:"-"`
	Username  string     `json:"username" gorm:"column:username;not null" validate:"min=1,max=32"`
	Password  string     `json:"password" gorm:"column:password;not null" binding:"required" validate:"min=5,max=128"`
	Phone     int64      `gorm:"column:phone" json:"phone"`
	Email     string     `gorm:"column:email" json:"email"`
	Avatar    string     `gorm:"column:avatar" json:"avatar"`
	CreatedAt time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt *time.Time `gorm:"column:deleted_at" json:"-"`
}

// Validate the fields.
func (u *UserBaseModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// UserInfo 对外暴露的结构体
type UserInfo struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Email    int    `json:"email"`
}

// TableName 表名
func (u *UserBaseModel) TableName() string {
	return "users"
}

// UserList 用户列表结构体
type UserList struct {
	Lock  *sync.Mutex
	IDMap map[uint64]*UserInfo
}

// Token represents a JSON web token.
type Token struct {
	Token string `json:"token"`
}

// Compare with the plain text password. Returns true if it's the same as the encrypted one (in the `User` struct).
func (u *UserBaseModel) Compare(pwd string) (err error) {
	err = auth.Compare(u.Password, pwd)
	return
}

// Encrypt the user password.
func (u *UserBaseModel) Encrypt() (err error) {
	u.Password, err = auth.Encrypt(u.Password)
	return
}
