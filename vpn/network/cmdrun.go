package network

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"log"
	"os"
	"os/exec"
)

func RunWithRollingOut(cmd *exec.Cmd, checker func(string) bool) (string, string, error) {
	log.Println(cmd.Args)
	var stdoutBuf, stderrBuf bytes.Buffer
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	go func() {
		_, _ = io.Copy(stdout, stdoutPipe)
	}()
	go func() {
		_, _ = io.Copy(stderr, stderrPipe)
	}()
	go func() {
		for {
			if (stdoutBuf.Len() > 0 && (checker(stdoutBuf.String()))) ||
				(stderrBuf.Len() > 0 && (checker(stderrBuf.String()))) {
				break
			}
		}
	}()
	if err := cmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		return stdoutBuf.String(), stderrBuf.String(), err
	}
	_ = cmd.Wait()
	if cmd.ProcessState.Success() {
		return stdoutBuf.String(), stderrBuf.String(), nil
	} else {
		return stdoutBuf.String(), stderrBuf.String(), errors.New("exit code is not 0")
	}
}
