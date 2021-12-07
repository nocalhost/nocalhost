package daemon_handler

import (
	"bufio"
	"fmt"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_server/command"
	"sync"
	"testing"
	//
	// Uncomment to load all auth plugins
	//_ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	//_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	//_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	//_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	//_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func TestName(t *testing.T) {
	reader, writer := io.Pipe()
	err := HandleSudoVPNOperate(&command.VPNOperateCommand{
		KubeConfig: clientcmd.RecommendedHomeFile,
		Namespace:  "nh90bwck",
		Resource:   "service/ratings",
		Action:     command.Connect,
	}, writer)
	fmt.Println(err)
	readers := bufio.NewReader(reader)
	for {
		line, _, err := readers.ReadLine()
		if err != nil {
			return
		} else {
			fmt.Println(string(line))
		}
	}
}

func TestMap(t *testing.T) {
	var a = sync.Map{}
	a.Store(ConnectInfo1{
		kubeconfigBytes: []byte("aa"),
		namespace:       "bb",
	}, "value_a")
	load, _ := a.Load(ConnectInfo1{
		kubeconfigBytes: []byte("aa"),
		namespace:       "bb",
	})
	fmt.Println(load)
}

type ConnectInfo1 struct {
	kubeconfigBytes []byte
	namespace       string
}
