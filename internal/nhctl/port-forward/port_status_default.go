// +build !windows

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
	result, err := tools.ExecCommand(nil, false, "bash", "-c", strings.Join(params, " "))
	if err != nil {
		log.Errorf("lsof error %s", err.Error())
	}
	if strings.ContainsAny(result, "LISTEN") {
		return "LISTEN"
	}
	return "CLOSE"
}
