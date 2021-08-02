/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package clientgoutils

type ErrorType string

const (
	InvalidYaml ErrorType = "InvalidYaml"
)

type TypedError struct {
	error
	ErrorType ErrorType
	Mes       string
}
