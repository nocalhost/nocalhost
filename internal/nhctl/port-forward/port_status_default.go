// +build !windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package port_forward

import (
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"strconv"
	"strings"
)

func PidPortStatus(pid int, port int) string {
	//portStatus := []string{
	//	"LISTEN",
	//	"ESTABLISHED",
	//	"FIN_WAIT_2",
	//	"CLOSE_WAIT",
	//	"TIME_WAIT",
	//	"FIN_WAIT_1",
	//	"LAST_ACK",
	//	"SYNC_RECEIVED",
	//	"SYNC_SEND",
	//}
	params := []string{
		"lsof",
		"-nP",
		"|",
		"grep",
		"LISTEN",
		"|",
		"grep",
		strconv.Itoa(pid),
		"|",
		"grep",
		strconv.Itoa(port),
	}
	result, err := tools.ExecCommand(nil, false, false, "bash", "-c", strings.Join(params, " "))
	if err != nil {
		log.Errorf("lsof error %s", err.Error())
	}
	if strings.ContainsAny(result, "LISTEN") {
		return "LISTEN"
	}
	return "CLOSE"
}
