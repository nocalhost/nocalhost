//go:build !windows
// +build !windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package wintun

import "github.com/pkg/errors"

func InstallWintunDriver() error {
	return errors.New("not implement")
}
