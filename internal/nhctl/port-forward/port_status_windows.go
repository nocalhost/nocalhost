// +build windows

/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package port_forward

import "fmt"

func PidPortStatus(pid int, port int) string {
	params := []string{
		"-aon",
		"|",
		"findstr",
		fmt.Sprintf("\"%d\"", pid),
		"|",
		"findstr",
		fmt.Sprintf("\"%d\"", port),
		"|",
		"findstr",
		fmt.Sprintf("\"%s\"", "LISTENING"),
	}
	result, err := tools.ExecCommand(nil, true, "netstat", params...)
	if err != nil {
		log.Errorf("netstat error %s", err.Error())
	}
	if strings.ContainsAny(result, "LISTENING") {
		return "LISTEN"
	}
	return "CLOSE"
}
