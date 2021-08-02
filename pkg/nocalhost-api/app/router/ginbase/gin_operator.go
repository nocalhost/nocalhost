/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package ginbase

import (
	"errors"
	"github.com/gin-gonic/gin"
)

const (
	NotExist = 0
)

func IsAdmin(c *gin.Context) bool {
	isAdmin, _ := c.Get("isAdmin")
	return isAdmin.(uint64) == 1
}

func LoginUser(c *gin.Context) (uint64, error) {
	userId, exists := c.Get("userId")
	if exists {
		return userId.(uint64), nil
	} else {
		return 0, errors.New("User not login ")
	}
}

func IsCurrentUser(c *gin.Context, userId uint64) bool {
	loginUserId, exists := c.Get("userId")
	if exists {
		return loginUserId == userId
	}

	return false
}
