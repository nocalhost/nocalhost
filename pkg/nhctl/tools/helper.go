/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package tools

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"math/rand"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
)

var logger *zap.Logger

//execute command
func ExecCommand(
	ctx context.Context, isDisplay bool, redirectStderr bool, ignoreCmdErr bool, commandName string, params ...string,
) (string, error) {
	var errStdout, errStderr error
	var result []byte

	var cmd *exec.Cmd
	if ctx == nil {
		cmd = exec.Command(commandName, params...)
	} else {
		//fmt.Println("command with ctx")
		cmd = exec.CommandContext(ctx, commandName, params...)
	}
	//log.Infof("Executing %s %v\n", commandName, params)
	cmdStr := []string{commandName}
	cmdStr = append(cmdStr, params...)
	if isDisplay {
		log.Infof("cmd: %s", strings.Join(cmdStr, " "))
	}
	// log.Info(cmd.Args)
	stdoutIn, err := cmd.StdoutPipe()
	stderrIn, err2 := cmd.StderrPipe()
	if err != nil || err2 != nil {
		return "", errors.New("error: command failed to get stdout pipe")
	}
	err = cmd.Start()
	if err != nil {
		return "", errors.Wrap(err, "Failed to start cmd")
	}

	go func() {
		result, errStdout = copyAndCapture(os.Stdout, stdoutIn, isDisplay)
	}()

	go func() {
		out := os.Stderr
		if redirectStderr {
			out = os.Stdout
		}
		_, errStderr = copyAndCapture(out, stderrIn, isDisplay)
	}()

	err = cmd.Wait()
	if !ignoreCmdErr && !cmd.ProcessState.Success() {
		return "", errors.Wrapf(err, "Error occur while exec command %v", cmdStr)
	}

	if errStderr != nil || errStdout != nil {
		log.Infof("%s %s", errStderr, errStdout)
		return "", errors.New("error occur when print")
	}

	return string(result), nil
}

func copyAndCapture(w io.Writer, r io.Reader, isDisplay bool) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			if isDisplay {
				w.Write(d)
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF || err == io.ErrClosedPipe || strings.Contains(err.Error(), "closed") {
				err = nil
			}
			return out, err
		}
	}
	// never reached
	panic(true)
	return nil, nil
}

// check kubectl and helm
func CheckThirdPartyCLI() (string, error) {
	kubectl := "kubectl"
	helm := "helm"
	if runtime.GOOS == "windows" {
		kubectl = "kubectl.exe"
		helm = "helm.exe"
	}
	_, err := exec.LookPath(kubectl)
	if err != nil {
		return "", err
	}
	_, err = exec.LookPath(helm)
	if err != nil {
		return "", err
	}
	return kubectl, nil
}

func GenerateRangeNum(min, max int) int {
	rand.Seed(time.Now().Unix())
	randNum := rand.Intn(max-min) + min
	return randNum
}

func RemoveDuplicateElement(languages []string) []string {
	result := make([]string, 0, len(languages))
	temp := map[string]struct{}{}
	for _, item := range languages {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
