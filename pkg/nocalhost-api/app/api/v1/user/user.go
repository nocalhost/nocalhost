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

// LoginCredentials 默认登录方式-邮箱
type LoginCredentials struct {
	Email    string `json:"email" form:"email"`
	Password string `json:"password" form:"password"`
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
