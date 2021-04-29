/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
