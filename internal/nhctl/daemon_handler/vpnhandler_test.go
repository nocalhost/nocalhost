package daemon_handler

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_server/command"
	"nocalhost/internal/nhctl/vpn/pkg"
	"nocalhost/internal/nhctl/vpn/util"
	"os"
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

func TestStruct(t *testing.T) {
	connected = &pkg.ConnectOptions{
		KubeconfigBytes: []byte("kube"),
		Namespace:       "ns",
	}
	vpnStatus, err := HandleVPNStatus()
	fmt.Println(err)
	marshal, err := json.Marshal(vpnStatus)
	fmt.Println(err)
	fmt.Println(string(marshal))
}

func TestGetDaemonVPNStatus(t *testing.T) {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		panic(err)
	}
	statusCommand, err := client.SendSudoVPNStatusCommand()
	if err != nil {
		panic(err)
	}
	marshal, err := json.Marshal(statusCommand)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(marshal))

}

func TestConnect(t *testing.T) {
	client, err := daemon_client.GetDaemonClient(true)
	if err != nil {
		panic(err)
	}
	logger := util.GetLoggerFromContext(context.TODO())
	options := pkg.ConnectOptions{
		KubeconfigPath: "/Users/naison/.kube/mesh",
		Namespace:      "naison-test",
	}
	if err = options.InitClient(context.TODO()); err != nil {
		panic(err)
	}
	if err = updateConnectConfigMap(options.GetClientSet().CoreV1().ConfigMaps(options.Namespace), insertFunc); err != nil {
		panic(err)
	}
	logger.Infof("connecting to new namespace: %s...", "naison-test")
	err = client.SendSudoVPNOperateCommand("/Users/naison/.kube/mesh",
		"naison-test",
		command.Connect,
		func(closer io.Reader) error {
			if ok := transStreamToWriter(closer, os.Stdout); !ok {
				panic(fmt.Errorf("failed to connect to namespace: %s", "naison-test"))
			}
			return nil
		})
	if err != nil {
		panic(err)
	}
}

func TestToStatus(t *testing.T) {
	ns := "demo"
	options := pkg.ConnectOptions{
		KubeconfigPath: "/Users/naison/.kube/nocalhost.large",
		Namespace:      ns,
	}
	if err := options.InitClient(context.TODO()); err != nil {
		panic(err)
	}
	get, err := options.GetClientSet().CoreV1().ConfigMaps(ns).
		Get(context.TODO(), util.TrafficManager, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	toStatus := ToStatus(get.Data)
	fmt.Println(toStatus)
}
