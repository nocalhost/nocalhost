package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInbound(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	initClient("")
	tomcat := "tomcat"
	ns := "test"
	port := "8080:8090"
	serviceName = &tomcat
	serviceNamespace = &ns
	portPair = &port
	privateKeyPath := filepath.Join(HomeDir(), ".nh", "ssh", "private", "key")
	Inbound(privateKeyPath)
}
