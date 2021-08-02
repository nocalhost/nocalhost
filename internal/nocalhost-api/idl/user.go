/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package idl

import (
	"nocalhost/internal/nocalhost-api/model"
)

// TransferUserInput
type TransferUserInput struct {
	CurUser *model.UserBaseModel
	User    *model.UserBaseModel
}

// TransferUser
func TransferUser(input *TransferUserInput) *model.UserInfo {
	if input.User == nil {
		return &model.UserInfo{}
	}

	return &model.UserInfo{
		ID:       input.User.ID,
		Username: input.User.Username,
		Avatar:   input.User.Avatar,
	}
}
