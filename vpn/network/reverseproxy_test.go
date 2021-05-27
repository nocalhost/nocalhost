package network

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInbound(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	initClient("")
	tomcat := "tomcat"
	ns := "test"
	port := "8080:8090"
	OPTION.serviceName = tomcat
	OPTION.serviceNamespace = ns
	OPTION.portPair = port
	privateKeyPath := filepath.Join(HomeDir(), ".nh", "ssh", "private", "key")
	Inbound(OPTION, privateKeyPath)
}

func TestCommand(t *testing.T) {
	cmd := exec.Command("kubectl", "get", "pods", "-w")
	c := make(chan struct{})
	go RunWithRollingOut(cmd, func(s string) bool {
		if strings.Contains(s, "tomcat") {
			c <- struct{}{}
			return true
		}
		return false
	})
	<-c
}

func TestSshS(t *testing.T) {
	cleanSsh()
}
