/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package application_user

type ApplicationUsersRequest struct {
	Users []int `json:"users" validate:"required"`
}
