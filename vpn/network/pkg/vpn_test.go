package pkg

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
	_ = os.Setenv("debug", "true")
	Start(OPTION)
}

func TestSsh(t *testing.T) {
	generateSshKey("", "")
}

func TestPortForward(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	initClient(OPTION)
	readyChan := make(chan struct{})
	stopsChan := make(chan struct{})
	err := portForwardPod("tomcat-shadow", "test", 5005, readyChan, stopsChan)
	if err != nil {
		log.Fatal(err)
	}
	<-readyChan
	fmt.Println("port forward ready")
	<-stopsChan
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
	err := PreCheck()
	clientgoutils.Must(err)
}

func TestDeleteDone(t *testing.T) {
	_ = os.Setenv("http_proxy", "")
	_ = os.Setenv("https_proxy", "")
	initClient(OPTION)
	OPTION.ServiceName = "tomcat"
	OPTION.ServiceNamespace = "test"
	OPTION.PortPair = "8080:8090"
	scaleDeploymentReplicasTo(OPTION, 0)
	//cleanShadow()
}
