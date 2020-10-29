package idl

import (
	"nocalhost/internal/nocalhost-api/model"
)

// TransferUserInput 转换输入字段
type TransferUserInput struct {
	CurUser *model.UserBaseModel
	User    *model.UserBaseModel
}

// TransferUser 对外暴露的user结构
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
