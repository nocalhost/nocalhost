package network

import (
	"context"
	"fmt"
	"k8s.io/utils/exec"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"testing"
)

func TestVpn(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	_ = os.Setenv("HOMEBREW_NO_AUTO_UPDATE", "true")
	Start()
}

func TestSsh(t *testing.T) {
	generateSSH("", "")
}

func TestSsh1(t *testing.T) {
	pair, s, err := MakeSSHKeyPair()

	fmt.Println(pair)
	fmt.Println(s)
	fmt.Println(err)
}

func TestPortForward(t *testing.T) {
	initClient("")
	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	forwarder, err := portForward("pods", "dnsserver-7bbc9d676f-b4rb5", 5000, readyChan, stopChan)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		if err = forwarder.ForwardPorts(); err != nil {
			panic(err)
		}
	}()
	<-readyChan
	fmt.Println("port forward ready")
	<-stopChan
}

func TestKubectl(t *testing.T) {
	kubeconfigPath := filepath.Join(HomeDir(), ".kube", "config")
	cmd := exec.New().
		CommandContext(
			context.TODO(),
			"kubectl",
			"port-forward",
			"service/dnsserver",
			"5000:22",
			"--namespace",
			"default",
			"--kubeconfig", kubeconfigPath)
	cmd.Start()
	err := cmd.Wait()
	if err != nil {
		log.Error(err)
	}
}

func TestInstall(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	_ = os.Setenv("HOMEBREW_NO_AUTO_UPDATE", "true")
	err := preCheck()
	clientgoutils.Must(err)
}
