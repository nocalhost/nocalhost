package util

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/util"
	"net"
	"testing"
)

var (
	namespace  string
	clientset  *kubernetes.Clientset
	restclient *rest.RESTClient
	config     *rest.Config
)

func TestShell(t *testing.T) {
	var err error

	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	configFlags.KubeConfig = &clientcmd.RecommendedHomeFile
	f := util.NewFactory(util.NewMatchVersionFlags(configFlags))

	if config, err = f.ToRESTConfig(); err != nil {
		log.Fatal(err)
	}
	if restclient, err = rest.RESTClientFor(config); err != nil {
		log.Fatal(err)
	}
	if clientset, err = kubernetes.NewForConfig(config); err != nil {
		log.Fatal(err)
	}
	if namespace, _, err = f.ToRawKubeConfigLoader().Namespace(); err != nil {
		log.Fatal(err)
	}

	out, err := Shell(clientset, restclient, config, TrafficManager, namespace, "cat /etc/resolv.conf | grep nameserver | awk '{print$2}'")
	serviceList, err := clientset.CoreV1().Services(v1.NamespaceSystem).List(context.Background(), v1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", "kube-dns").String(),
	})

	fmt.Println(out == serviceList.Items[0].Spec.ClusterIP)
}

func TestDeleteRule(t *testing.T) {
	DeleteWindowsFirewallRule()
}

func TestUDP(t *testing.T) {
	relay, err := net.ListenUDP("udp", &net.UDPAddr{Port: 12345})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(relay.LocalAddr())
	fmt.Println(relay.RemoteAddr())
}
