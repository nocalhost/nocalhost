//go:build !windows
// +build !windows

/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func RunWithElevated() error {
	cmd := exec.Command("sudo", "-p", "Password:", "-S", os.Args[0], os.Args[1], "elevate")
	//log.Info(cmd.Args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	// while send single CRTL+C, command will quit immediately, but output will cut off and print util quit final
	// so, mute single CTRL+C, let inner command handle single only
	go func() {
		signals := make(chan os.Signal)
		signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGSTOP)
		<-signals
	}()
	return cmd.Run()
}

func IsAdmin() bool {
	return os.Getuid() == 0
}
