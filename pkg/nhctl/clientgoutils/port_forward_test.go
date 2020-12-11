package clientgoutils

import (
	"fmt"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"testing"
	"time"
)

func TestClientGoUtils_PortForward(t *testing.T) {
	//utils.GetHomePath()
	client, err := NewClientGoUtils(filepath.Join(utils.GetHomePath(), ".kube", "admin-config"), time.Minute)
	if err != nil {
		panic(err)
	}

	fps := []*ForwardPort{{LocalPort: 11223, RemotePort: 22}}

	pf, err := client.CreatePortForwarder("demo30", "details-77cc4f49fd-nl7kn", fps)
	if err != nil {
		panic(err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- pf.ForwardPorts()
	}()

	for {
		select {
		case <-pf.Ready:
			fmt.Println("port forward is ready")
		case <-errChan:
			fmt.Println("err")
		}
	}

	time.Sleep(time.Minute)
	pf.Close()

}
