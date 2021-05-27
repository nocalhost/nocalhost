package network

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// multiple remote service port
func sshReverseProxy(remotePort, localPort string, sshLocalPort int, privateKeyPath string, wg *sync.WaitGroup) {
	log.Println("prepare to start reverse proxy")
	cmd := exec.Command("ssh", "-NR",
		fmt.Sprintf("0.0.0.0:%s:127.0.0.1:%s", remotePort, localPort),
		"root@127.0.0.1", "-p", strconv.Itoa(sshLocalPort),
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-i",
		privateKeyPath)
	stdout, stderr, err := RunWithRollingOut(cmd, func(s string) bool {
		if strings.Contains(s, "Permanently added") {
			wg.Done()
			return true
		}
		return false
	})
	if err != nil {
		log.Printf("start reverse proxy failed, error: %v, stdout: %s, stderr: %s", err, stdout, stderr)
		stopChan <- syscall.SIGQUIT
		return
	} else {
		log.Println("stdout: " + stdout)
		log.Println("stderr: " + stderr)
	}
}

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
