package pkg

import (
	"context"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

func sshuttleOutbound(serviceIp, sshPrivateKeyPath string, localSshPort int, cidrs []string, c chan struct{}) {
	args := []string{
		"-r",
		"root@127.0.0.1:" + strconv.Itoa(localSshPort),
		"-e", "ssh -oStrictHostKeyChecking=no -oUserKnownHostsFile=/dev/null -i " + sshPrivateKeyPath,
		"-x",
		"127.0.0.1",
		"--dns",
		"--to-ns",
		serviceIp,
	}
	args = append(args, cidrs...)
	cmd := exec.CommandContext(context.TODO(), "sshuttle", args...)
	out, s, err := RunWithRollingOut(cmd, func(s string) bool {
		if strings.Contains(s, "Connected to server") {
			c <- struct{}{}
			return true
		}
		return false
	})
	if err != nil {
		log.Printf("error: %v, stdout: %s, stderr: %s", err, out, s)
	}
}
