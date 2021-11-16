//go:build !windows
// +build !windows

package util

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func RunWithElevated() {
	cmd := exec.Command("sudo", os.Args...)
	log.Info(cmd.Args)
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
	err := cmd.Run()
	if err != nil {
		log.Warn(err)
	}
}

func IsAdmin() bool {
	return os.Getuid() == 0
}
