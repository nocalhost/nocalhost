package daemon_handler

import (
	"bufio"
	"fmt"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_server/command"
	"testing"
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
