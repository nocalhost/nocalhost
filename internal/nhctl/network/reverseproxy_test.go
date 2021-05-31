package network

import (
	"nocalhost/internal/nhctl/model"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInbound(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	initClient(&model.Option)
	tomcat := "tomcat"
	ns := "test"
	port := "8080:8090"
	model.Option.ServiceName = tomcat
	model.Option.Namespace = ns
	model.Option.PortPairs = port
	privateKeyPath := filepath.Join(HomeDir(), ".nh", "ssh", "private", "key")
	Inbound(model.Option, privateKeyPath)
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
