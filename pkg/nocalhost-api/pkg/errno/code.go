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

package errno

//nolint: golint
var (
	// Common errors
	OK                  = &Errno{Code: 0, Message: "OK"}
	InternalServerError = &Errno{Code: 10001, Message: "Internal server error"}
	ErrBind             = &Errno{Code: 10002, Message: "Error occurred while binding the request body to the struct."}
	ErrParam            = &Errno{Code: 10003, Message: "参数有误"}
	ErrSignParam        = &Errno{Code: 10004, Message: "签名参数有误"}

	// user errors
	ErrUserNotFound          = &Errno{Code: 20102, Message: "The user was not found."}
	ErrTokenInvalid          = &Errno{Code: 20103, Message: "token 无效或登陆过期，请重新登陆"}
	ErrEmailOrPassword       = &Errno{Code: 20111, Message: "邮箱或密码错误"}
	ErrTwicePasswordNotMatch = &Errno{Code: 20112, Message: "两次密码输入不一致"}
	ErrRegisterFailed        = &Errno{Code: 20113, Message: "注册失败"}
	ErrUserNotAllow          = &Errno{Code: 20114, Message: "用户被禁用"}
	ErrCreateUserDenied      = &Errno{Code: 20114, Message: "无创建用户权限"}
	ErrUpdateUserDenied      = &Errno{Code: 20114, Message: "无修改用户权限"}
	ErrDeleteUser            = &Errno{Code: 20115, Message: "删除用户失败"}

	// cluster errors
	ErrClusterCreate = &Errno{Code: 30100, Message: "添加集群失败，请重试"}

	// application errors
	ErrApplicationCreate      = &Errno{Code: 40100, Message: "添加应用失败，请重试"}
	ErrApplicationGet         = &Errno{Code: 40101, Message: "获取应用失败，请重试"}
	ErrApplicationDelete      = &Errno{Code: 40102, Message: "删除应用失败，请重试"}
	ErrApplicationUpdate      = &Errno{Code: 40103, Message: "更新应用失败，请重试"}
	ErrBindApplicationClsuter = &Errno{Code: 40104, Message: "绑定集群失败，请重试"}
	ErrPermissionApplication  = &Errno{Code: 40105, Message: "无此应用权限"}
	ErrPermissionCluster      = &Errno{Code: 40106, Message: "无此集群权限"}
)
