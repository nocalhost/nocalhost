package clientgoutils

import (
	"fmt"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/utils"
	"os"
	"path/filepath"
	"testing"
)

func TestClientGoUtils_PortForward(t *testing.T) {
	//utils.GetHomePath()
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube", "devpool"), "")
	if err != nil {
		panic(err)
	}

	fps := []*ForwardPort{{LocalPort: 39080, RemotePort: 9080}}

	pf, err := client.CreatePortForwarder("authors-788bb4cf5c-76968", fps, nil, nil, genericclioptions.IOStreams{
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
		case <-errChan:
			fmt.Println("err")
		}
	}
}
