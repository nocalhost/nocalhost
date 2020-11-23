/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package user

import (
	"nocalhost/internal/nocalhost-api/model"
)

// CreateRequest 创建用户请求
type CreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateResponse 创建用户响应
type CreateResponse struct {
	Username string `json:"username"`
}

// RegisterRequest 注册
type RegisterRequest struct {
	Email           string `json:"email" form:"email"`
	Password        string `json:"password" form:"password"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password"`
}

// CreateUserRequest 添加用户
type CreateUserRequest struct {
	Email           string  `json:"email" form:"email" binding:"required"`
	Name            string  `json:"name" form:"name" binding:"required"`
	Password        string  `json:"password" form:"password" binding:"required"`
	ConfirmPassword string  `json:"confirm_password" form:"confirm_password" binding:"required"`
	Status          *uint64 `json:"status" form:"status" binding:"required"`
}

// UpdateUserRequest 更新用户
type UpdateUserRequest struct {
	Email    string  `json:"email" form:"email" binding:""`
	Name     string  `json:"name" form:"name" binding:""`
	Password string  `json:"password" form:"password" binding:""`
	Status   *uint64 `json:"status" form:"status" binding:"required"`
}

// LoginCredentials 默认登录方式-邮箱
type LoginCredentials struct {
	Email    string `json:"email" form:"email" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
	From     string `json:"from" form:"from" example:"web 端不传该字段"`
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	Avatar string `json:"avatar"`
	Sex    int    `json:"sex"`
}

// ListResponse 通用列表resp
type ListResponse struct {
	TotalCount uint64      `json:"total_count"`
	HasMore    int         `json:"has_more"`
	PageKey    string      `json:"page_key"`
	PageValue  int         `json:"page_value"`
	Items      interface{} `json:"items"`
}

// SwaggerListResponse 文档
type SwaggerListResponse struct {
	TotalCount uint64           `json:"totalCount"`
	UserList   []model.UserInfo `json:"userList"`
}
