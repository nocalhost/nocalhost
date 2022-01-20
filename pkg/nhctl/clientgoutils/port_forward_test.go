package clientgoutils

import (
	"fmt"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/utils"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientGoUtils_PortForward(t *testing.T) {
	//utils.GetHomePath()
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube", "socat"), "default")
	if err != nil {
		panic(err)
	}

	fps := []*ForwardPort{{LocalPort: 39080, RemotePort: 9080}}

	pf, err := client.CreatePortForwarder("productpage-5f5f74f96b-hm2ct", fps, nil, nil, genericclioptions.IOStreams{
		Out: os.Stdout,
	})
	if err != nil {
		panic(err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- pf.ForwardPorts()
	}()
	defer pf.Close()
	for {
		select {
		case <-pf.Ready:
			//fmt.Println("port forward is ready")
		case err = <-errChan:
			fmt.Printf("%v, err: %v", strings.Contains(err.Error(), "failed to find socat"), err)
			panic(err)
		}
	}
}

func TestName(t *testing.T) {
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube", "socat"), "default")
	if err != nil {
		panic(err)
	}
	err = client.ForwardPortForwardByPod(
		"productpage-5f5f74f96b-hm2ct",
		39080,
		9080,
		make(chan struct{}, 1),
		make(chan struct{}, 1),
		genericclioptions.IOStreams{
			Out: os.Stdout,
		},
	)
	if err != nil {
		fmt.Printf("%v, err: %v", strings.Contains(err.Error(), "failed to find socat"), err)
		panic(err)
	}
}
