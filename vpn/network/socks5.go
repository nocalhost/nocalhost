package network

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func sshOutbound(privateKeyPath string, sock5Port, localSshPort int, c chan struct{}) {
	cmd := exec.Command("ssh", "-ND",
		"0.0.0.0:"+strconv.Itoa(sock5Port),
		"root@127.0.0.1",
		"-p", strconv.Itoa(localSshPort),
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-i", privateKeyPath)
	stdout, stderr, err := RunWithRollingOut(cmd, func(s string) bool {
		if strings.Contains(s, "Permanently added") {
			c <- struct{}{}
			return true
		}
		return false
	})
	if err != nil {
		log.Printf("ssh -d err: %v, stdout: %s, stderr: %s\n", err, stdout, stderr)
	}
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
