//go:build windows
// +build windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package pkg

import (
	"net"
)

func getRouteTable() (map[string][]*net.IPNet, error) {
	return make(map[string][]*net.IPNet), nil
}

func disableDevice(list []string) error {
	return nil
}
