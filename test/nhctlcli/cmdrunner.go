/*
Copyright 2021 The Nocalhost Authors.
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

package nhctlcli

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

var Runner = &CmdRunner{}

type CmdRunner struct{}

func (r *CmdRunner) RunPanicIfError(cmd *exec.Cmd) {
	if stdout, stderr, err := r.Run(cmd); err != nil {
		panic(fmt.Sprintf("Run command: %s, error: %v, stdout: %s, stderr: %s\n", cmd.Args, err, stdout, stderr))
	}
}

func (r *CmdRunner) CheckResult(cmd *exec.Cmd, stdout string, stderr string, err error) {
	if err != nil {
		panic(fmt.Sprintf("Run command: %s, error: %v, stdout: %s, stderr: %s\n", cmd.Args, err, stdout, stderr))
	}
}

func (r *CmdRunner) Run(cmd *exec.Cmd) (string, string, error) {
	fmt.Printf("Running command: %s\n", cmd.Args)

	stdout := bytes.Buffer{}
	cmd.Stdout = &stdout
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("starting command %v: %w", cmd, err)
	}

	_ = cmd.Wait()

	if cmd.ProcessState.Success() {
		return stdout.String(), stderr.String(), nil
	} else {
		return stdout.String(), stderr.String(), fmt.Errorf(
			"Execute command: %v, stdout: %v, stderr: %v\n", cmd.Args, stdout.String(), stderr.String())
	}
}

func (r *CmdRunner) RunWithRollingOut(cmd *exec.Cmd) (string, string, error) {
	fmt.Printf("Running command: %s\n", cmd.Args)
	var stdoutLog strings.Builder
	var stderrLog strings.Builder
	stdoutPipe, err := cmd.StdoutPipe()
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return stdoutLog.String(), stderrLog.String(), err
	}
	if err = cmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		return "", "", err
	}
	defer stdoutPipe.Close()
	defer stderrPipe.Close()
	stdoutReader := bufio.NewReaderSize(stdoutPipe, 1024)
	stderrReader := bufio.NewReaderSize(stderrPipe, 1024)
	go func() {
		for {
			line, isPrefix, err := stdoutReader.ReadLine()
			if err != nil {
				if err != io.EOF && !strings.Contains(err.Error(), "closed") {
					os.Stdout.WriteString(fmt.Sprintf("command log error: %v, log: %v\n", err, string(line)))
				}
				break
			}
			if len(line) != 0 && !isPrefix {
				os.Stdout.WriteString(string(line) + "\n")
				stdoutLog.WriteString(string(line) + "\n")
			}
		}
	}()
	go func() {
		for {
			line, isPrefix, err := stderrReader.ReadLine()
			if err != nil {
				if err != io.EOF && !strings.Contains(err.Error(), "closed") {
					os.Stderr.WriteString(fmt.Sprintf("command log error: %v, log: %v\n", err, string(line)))
				}
				break
			}
			if len(line) != 0 && !isPrefix {
				os.Stderr.WriteString(string(line) + "\n")
				stderrLog.WriteString(string(line) + "\n")
			}
		}
	}()
	_ = cmd.Wait()
	time.Sleep(2 * time.Second)
	if cmd.ProcessState.Success() && stderrLog.Len() == 0 {
		return stdoutLog.String(), "", nil
	} else {
		return stdoutLog.String(), stderrLog.String(), errors.New("exit code is not 0")
	}
}
