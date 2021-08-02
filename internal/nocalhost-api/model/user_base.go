/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package model

import (
	"math/rand"
	"time"

	"nocalhost/pkg/nocalhost-api/pkg/auth"

	validator "github.com/go-playground/validator/v10"
)

// UserBaseModel
type UserBaseModel struct {
	ID           uint64     `gorm:"primary_key;AUTO_INCREMENT;column:id" json:"id"`
	Uuid         string     `gorm:"column:uuid;not null" json:"-"`
	SaName       string     `gorm:"column:sa_name" json:"sa_name"`
	Name         string     `json:"name" gorm:"column:name;not null" json:"name"`
	Username     string     `json:"username" gorm:"column:username;not null" validate:"min=1,max=32"`
	Password     string     `json:"-" gorm:"column:password;not null" binding:"required" validate:"min=5,max=128"`
	Phone        int64      `gorm:"column:phone" json:"phone"`
	Email        string     `gorm:"column:email" json:"email"`
	IsAdmin      *uint64    `gorm:"column:is_admin" json:"is_admin"`
	Status       *uint64    `gorm:"column:status" json:"status"`
	ClusterAdmin *uint64    `gorm:"column:cluster_admin" json:"cluster_admin"`
	Avatar       string     `gorm:"column:avatar" json:"avatar"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"-"`
	UpdatedAt    time.Time  `gorm:"column:updated_at" json:"-"`
	DeletedAt    *time.Time `gorm:"column:deleted_at" json:"-"`
}

// Validate the fields.
func (u *UserBaseModel) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
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
