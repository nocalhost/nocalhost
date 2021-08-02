// +build windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package port_forward

import (
	"fmt"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"strings"
)

func PidPortStatus(pid int, port int) string {
	result, err := tools.ExecCommand(nil, false, false, "cmd", "/c", fmt.Sprintf("netstat -ano | findstr %d | findstr %d | findstr LISTENING", pid, port))
	if err != nil {
		log.Errorf("netstat error %s", err.Error())
	}
	if strings.ContainsAny(result, "LISTENING") {
		return "LISTEN"
	}
	return "CLOSE"
}
