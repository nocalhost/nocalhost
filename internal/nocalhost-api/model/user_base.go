/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package model

import (
	"math/rand"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/pkg/nhctl/log"
	"time"

	"nocalhost/pkg/nocalhost-api/pkg/auth"

	"github.com/go-playground/validator/v10"
)

// UserBaseModel
type UserBaseModel struct {
	ID       uint64 `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Uuid     string `gorm:"column:uuid;not null" json:"-"`
	SaName   string `gorm:"column:sa_name" json:"sa_name"`
	Name     string `json:"name" gorm:"column:name;not null" json:"name"`
	Username string `json:"username" gorm:"column:username;not null" validate:"min=1,max=32"`
	Password string `json:"-" gorm:"column:password;not null" binding:"required" validate:"min=5,max=128"`
	Phone    int64  `gorm:"column:phone" json:"phone"`
	Email    string `gorm:"column:email" json:"email"`

	LdapDN  string `gorm:"column:ldap_dn" json:"ldap_dn"`
	LdapGen uint64 `gorm:"column:ldap_gen" json:"ldap_gen"`

	IsAdmin      *uint64    `gorm:"column:is_admin" json:"is_admin"`
	Status       *uint64    `gorm:"column:status" json:"status"`
	ClusterAdmin *uint64    `gorm:"column:cluster_admin" json:"cluster_admin"`
	Avatar       string     `gorm:"column:avatar" json:"avatar"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt    *time.Time `gorm:"column:deleted_at" json:"-"`
}

func (u *UserBaseModel) NeedToUpdateProfileInLdap(userName, ldapDN string, admin bool) bool {

	// admin sign no change
	bool := (u.IsAdmin != nil && *u.IsAdmin != *_const.BoolToUint64Pointer(admin)) ||
		u.IsAdmin == nil ||

		// dn no change
		u.LdapDN != ldapDN ||

		// status is enabled
		(u.Status != nil && *u.Status != *_const.BoolToUint64Pointer(true)) ||
		u.Status == nil ||

		// username is specified and no change
		// or username is not specified and username is set before
		(userName != "" && u.Name != userName)

	if bool {
		log.Error("")
	}

	return bool
}

// Validate the fields.
func (u *UserBaseModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}

// Validate the fields.
func (u *UserBaseModel) ToUserSimple() *UserSimple {
	return &UserSimple{
		ID:           u.ID,
		Name:         u.Name,
		Username:     u.Username,
		Phone:        u.Phone,
		Email:        u.Email,
		IsAdmin:      u.IsAdmin,
		Status:       u.Status,
		ClusterAdmin: u.ClusterAdmin,
		Avatar:       u.Avatar,
		CreatedAt:    u.CreatedAt,
	}
}

type UserSimple struct {
	ID           uint64    `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Name         string    `json:"name" gorm:"column:name;not null" json:"name"`
	Username     string    `json:"username" gorm:"column:username;not null" validate:"min=1,max=32"`
	Phone        int64     `gorm:"column:phone" json:"phone"`
	Email        string    `gorm:"column:email" json:"email"`
	IsAdmin      *uint64   `gorm:"column:is_admin" json:"is_admin"`
	Status       *uint64   `gorm:"column:status" json:"status"`
	ClusterAdmin *uint64   `gorm:"column:cluster_admin" json:"cluster_admin"`
	Avatar       string    `gorm:"column:avatar" json:"avatar"`
	CreatedAt    time.Time `gorm:"column:created_at" json:"-"`
}

// UserInfo
type UserInfo struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Email    int    `json:"email"`
	Status   uint64 `json:"status"`
}

// UserList
type UserList struct {
	ID           uint64 `gorm:"column:id" json:"id"`
	Name         string `gorm:"column:name" json:"name"`
	SaName       string `gorm:"column:sa_name;not null" json:"sa_ame"`
	Email        string `gorm:"column:email" json:"email"`
	ClusterCount uint64 `gorm:"column:cluster_count" json:"cluster_count"`
	Status       uint64 `gorm:"column:status" json:"status"`
	IsAdmin      uint64 `gorm:"column:is_admin" json:"is_admin"`
}

// TableName
func (u *UserBaseModel) TableName() string {
	return "users"
}

// Token represents a JSON web token.
type Token struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
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

// serviceaccount must match DNS-1123 label, capital doesn't allow
func GenerateSaName() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return "nhsa" + string(b)
}
