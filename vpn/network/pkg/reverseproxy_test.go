package pkg

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
	initClient(&Option)
	tomcat := "tomcat"
	ns := "test"
	port := "8080:8090"
	Option.ServiceName = tomcat
	Option.Namespace = ns
	Option.PortPairs = port
	privateKeyPath := filepath.Join(HomeDir(), ".nh", "ssh", "private", "key")
	Inbound(Option, privateKeyPath)
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
