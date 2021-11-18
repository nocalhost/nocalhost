/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package runner

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"strings"
)

var Runner = &CmdRunner{}

type CmdRunner struct{}

func (r *CmdRunner) RunWithCheckResult(suitName string, cmd *exec.Cmd) error {
	if stdout, stderr, err := r.Run(suitName, cmd); err != nil {
		return errors.Errorf(
			"Run command: %s, error: %v, stdout: %s, stderr: %s", cmd.Args, err, stdout, stderr,
		)
	}
	return nil
}

func (r *CmdRunner) CheckResult(cmd *exec.Cmd, stdout string, stderr string, err error) error {
	if err != nil {
		return errors.Errorf(
			"Run command: %s, error: %v, stdout: %s, stderr: %s", cmd.Args, err, stdout, stderr,
		)
	}
	return nil
}

func (r *CmdRunner) RunSimple(suitName string, cmd *exec.Cmd, ignoreStdErr bool, stdoutConsumer func(string) error) error {
	stdout, stderr, err := r.Run(suitName, cmd)

	if err != nil {
		return err
	}
	if !ignoreStdErr && stderr != "" {
		return errors.New(stderr)
	}

	return stdoutConsumer(stdout)
}

func (r *CmdRunner) Run(suitName string, cmd *exec.Cmd) (string, string, error) {
	log.TestLogger(suitName).Infof("Running command: %s", cmd.Args)

	stdout := bytes.Buffer{}
	cmd.Stdout = &stdout
	stderr := bytes.Buffer{}
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", "", errors.Errorf("starting command %v: %v", cmd, err)
	}

	if err := cmd.Wait(); err != nil {
		return stdout.String(), stderr.String(), errors.Wrap(err, "")
	}

	if stderr.Len() > 0 {
		log.TestLogger(suitName).Infof("Command output: [\n%s\n], stderr: [\n%s\n]", stdout.String(), stderr.String())
	}

	return stdout.String(), stderr.String(), nil
}

func (r *CmdRunner) RunWithRollingOutWithChecker(suitName string, cmd *exec.Cmd, checker func(log string) bool) (string, string, error) {
	logger := log.TestLogger(suitName)
	logger.Infof("Running command: %s", cmd.Args)

	stdoutBuf := bytes.NewBuffer(make([]byte, 0))
	stderrBuf := bytes.NewBuffer(make([]byte, 0))

	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()
	stdout := io.MultiWriter(os.Stdout, stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, stderrBuf)
	go func() {
		_, _ = io.Copy(stdout, stdoutPipe)
	}()
	go func() {
		_, _ = io.Copy(stderr, stderrPipe)
	}()
	go func() {
		if checker != nil {
			for {
				if checker(stdoutBuf.String()) || checker(stderrBuf.String()) {
					break
				}
			}
		}
	}()
	if err := cmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		return stdoutBuf.String(), stderrBuf.String(), err
	}
	_ = cmd.Wait()
	var err error
	if !cmd.ProcessState.Success() {
		err = errors.New("exit code is not 0")
	}

	stdoutStr := strings.TrimSpace(stdoutBuf.String())
	stderrStr := strings.TrimSpace(stderrBuf.String())

	if stderrStr == "" {
		logger.Infof("Command %s \n[INFO]stdout:\n%s", cmd.Args, stdoutStr)
	} else {
		logger.Infof("Command %s \n[INFO]stdout:\n%s\n[INFO]stderr:%s", cmd.Args, stdoutStr, stderrStr)
	}

	return stdoutStr, stderrStr, err
}
