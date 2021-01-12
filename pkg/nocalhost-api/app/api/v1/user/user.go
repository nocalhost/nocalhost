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

// CreateRequest
type CreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateResponse
type CreateResponse struct {
	Username string `json:"username"`
}

// RegisterRequest
type RegisterRequest struct {
	Email           string `json:"email" form:"email"`
	Password        string `json:"password" form:"password"`
	ConfirmPassword string `json:"confirm_password" form:"confirm_password"`
}

// CreateUserRequest
type CreateUserRequest struct {
	Email           string  `json:"email" form:"email" binding:"required"`
	Name            string  `json:"name" form:"name" binding:"required"`
	Password        string  `json:"password" form:"password" binding:"required"`
	ConfirmPassword string  `json:"confirm_password" form:"confirm_password" binding:"required"`
	Status          *uint64 `json:"status" form:"status" binding:"required"`
	IsAdmin         *uint64 `json:"is_admin" form:"is_admin" binding:"required"`
}

// UpdateUserRequest
type UpdateUserRequest struct {
	Email    string  `json:"email" form:"email"`
	Name     string  `json:"name" form:"name"`
	Password string  `json:"password" form:"password"`
	Status   *uint64 `json:"status" form:"status"`
	IsAdmin  *uint64 `json:"is_admin" form:"is_admin"`
}

// LoginCredentials
type LoginCredentials struct {
	Email    string `json:"email" form:"email" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
	From     string `json:"from" form:"from" example:"only use for plugin, web interface do not send this key"`
}

// UpdateRequest
type UpdateRequest struct {
	Avatar string `json:"avatar"`
	Sex    int    `json:"sex"`
}

// ListResponse
type ListResponse struct {
	TotalCount uint64      `json:"total_count"`
	HasMore    int         `json:"has_more"`
	PageKey    string      `json:"page_key"`
	PageValue  int         `json:"page_value"`
	Items      interface{} `json:"items"`
}

// SwaggerListResponse
type SwaggerListResponse struct {
	TotalCount uint64           `json:"totalCount"`
	UserList   []model.UserInfo `json:"userList"`
}
