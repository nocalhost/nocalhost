package daemon_handler

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/util"
	"sync"
	"testing"
	"time"
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

func TestVPNStatus(t *testing.T) {
	file, _ := ioutil.ReadFile(clientcmd.RecommendedHomeFile)
	GetOrGenerateConfigMapWatcher(file, "naison", nil)
	time.Sleep(time.Second * 5)
	load, _ := GetReverseInfo().Load(util.GenerateKey(file, "naison"))
	resources := load.(*status).reverse.GetBelongToMeResources().List()
	for {
		time.Sleep(time.Second * 1)
		GetReverseInfo().Range(func(key, value interface{}) bool {
			value.(*status).reverse.GetBelongToMeResources().ForEach(func(k string, v *resourceInfo) {
				fmt.Printf("%s is %s\n", k, v.Health)
				return
			})
			meResources := value.(*status).reverse.GetBelongToMeResources().List()
			if len(resources) != len(meResources) {
				fmt.Println("nooooo size")
			}
			for i := 0; i < len(resources); i++ {
				if resources[i] != meResources[i] {
					fmt.Println("nooooo")
				}
			}
			return true
		})
	}
}

func TestMaps(t *testing.T) {
	var m sync.Map
	m.LoadOrStore("1", &a{})
	m.Range(func(key, value interface{}) bool {
		value.(*a).name = "bbbbbbbbb"
		return true
	})
	l, _ := m.Load("1")
	fmt.Println(l)
}

type a struct {
	name string
}
