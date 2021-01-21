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

import (
	"fmt"
	"nocalhost/pkg/nhctl/log"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

const (
	winPre   = "cmd /c "
	LinuxPre = "bash /c "
)

func PidPortStatus(pid int, port int) string {
	//params := []string{
	//	"netstat",
	//	"-aon",
	//	"|",
	//	"findstr",
	//	fmt.Sprintf("%d", pid),
	//	"|",
	//	"findstr",
	//	fmt.Sprintf("%d", port),
	//	"|",
	//	"findstr",
	//	fmt.Sprintf("%s", "LISTENING"),
	//}
	cmd := `"netstat -ano | findstr ` + strconv.Itoa(pid) + `"`
	result, err := RunCmd(cmd)
	// result, err := tools.ExecCommand(nil, true, "cmd", "/c", fmt.Sprintf("netstat -ano | findstr %d | findstr %d | findstr LISTENING", pid, port))
	if err != nil {
		log.Errorf("netstat error %s", err.Error())
	}
	fmt.Println(string(result))
	if strings.ContainsAny(string(result), "LISTENING") {
		return "LISTEN"
	}
	return "CLOSE"
}

func RunCmd(cmd string) (result []byte, err error) {
	cmdPip := strings.Split(cmd, "|")
	if len(cmdPip) < 2 {
		return run(cmd).Output()
	}

	return runPipe(cmdPip)
}

func run(cmd string) *exec.Cmd {
	if strings.Contains(cmd, "findstr") || !strings.Contains(cmd, "find") {
		cmd = strings.Replace(cmd, `"`, "", -1)
		cmd = strings.TrimSpace(cmd)
	}
	cmdList := strings.Split(cmd, " ")

	return exec.Command(cmdList[0], cmdList[1:]...)
}

func runPipe(pip []string) (result []byte, err error) {
	var cmds []*exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmds = append(cmds, run(winPre+pip[0]))
	case "linux":
		cmds = append(cmds, run(LinuxPre+pip[0]))
	default:
		return nil, fmt.Errorf("Not supported by the system. %v", runtime.GOOS)
	}

	for i := 1; i < len(pip); i++ {
		cmds = append(cmds, run(pip[i]))
		cmds[i].Stdin, err = cmds[i-1].StdoutPipe()
		if err != nil {
			return nil, err
		}
	}

	end := len(cmds) - 1
	stdout, _ := cmds[end].StdoutPipe()

	for i := end; i > 0; i-- {
		cmds[i].Start()
	}
	cmds[0].Run()

	buf := make([]byte, 512)
	n, _ := stdout.Read(buf)

	err = cmds[end].Wait()

	return buf[:n], err
}
