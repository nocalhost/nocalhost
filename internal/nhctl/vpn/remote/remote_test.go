/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	json2 "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net"
	"nocalhost/internal/nhctl/vpn/util"
	"strings"
	"testing"
	"time"
)

//func TestCreateServer(t *testing.T) {
//	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
//		&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientcmd.RecommendedHomeFile}, nil,
//	)
//	config, err := clientConfig.ClientConfig()
//	if err != nil {
//		log.Fatal(err)
//	}
//	clientset, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	i := &net.IPNet{
//		IP:   net.ParseIP("192.168.254.100"),
//		Mask: net.IPv4Mask(255, 255, 255, 0),
//	}
//
//	j := &net.IPNet{
//		IP:   net.ParseIP("172.20.0.0"),
//		Mask: net.IPv4Mask(255, 255, 0, 0),
//	}
//
//	server, err := pkg.CreateOutboundRouterPod(clientset, "test", i, []*net.IPNet{j})
//	fmt.Println(server)
//}

func TestGetIp(t *testing.T) {
	ip := &net.IPNet{
		IP:   net.IPv4(192, 168, 254, 100),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	fmt.Println(ip.String())
}

func TestGetIPFromDHCP(t *testing.T) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientcmd.RecommendedHomeFile}, nil,
	)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	manager := NewDHCPManager(clientset, "test", nil)
	manager.InitDHCPIfNecessary(context.TODO())
	for i := 0; i < 10; i++ {
		ipNet, err := manager.RentIP(true)
		ipNet2, err := manager.RentIP(true)
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			fmt.Printf("%s->%s\n", ipNet.String(), ipNet2.String())
		}
		time.Sleep(time.Millisecond * 10)
		err = manager.ReleaseIP(int(ipNet.IP[3]))
		err = manager.ReleaseIP(int(ipNet2.IP[3]))
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func TestOwnerRef(t *testing.T) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientcmd.RecommendedHomeFile}, nil,
	)
	config, _ := clientConfig.ClientConfig()
	clientset, _ := kubernetes.NewForConfig(config)
	//get, _ := clientset.CoreV1().Pods("test").Get(context.Background(), "tomcat-7449544d95-nv7gr", metav1.GetOptions{})
	get, _ := clientset.CoreV1().Pods("test").Get(context.Background(), "mysql-0", metav1.GetOptions{})

	of := metav1.GetControllerOf(get)
	for of != nil {
		b, err := clientset.AppsV1().RESTClient().Get().Namespace("test").
			Name(of.Name).Resource(strings.ToLower(of.Kind) + "s").Do(context.Background()).Raw()
		if k8serrors.IsNotFound(err) {
			return
		}
		var replicaSet v1.ReplicaSet
		if err = json.Unmarshal(b, &replicaSet); err == nil && len(replicaSet.Name) != 0 {
			fmt.Printf("%s-%s\n", replicaSet.Kind, replicaSet.Name)
			of = metav1.GetControllerOfNoCopy(&replicaSet)
			continue
		}
		var statefulSet v1.StatefulSet
		if err = json.Unmarshal(b, &statefulSet); err == nil && len(statefulSet.Name) != 0 {
			fmt.Printf("%s-%s\n", statefulSet.Kind, statefulSet.Name)
			of = metav1.GetControllerOfNoCopy(&statefulSet)
			continue
		}
		var deployment v1.Deployment
		if err = json.Unmarshal(b, &deployment); err == nil && len(deployment.Name) != 0 {
			fmt.Printf("%s-%s\n", deployment.Kind, deployment.Name)
			of = metav1.GetControllerOfNoCopy(&deployment)
			continue
		}
	}
}

func TestGet(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	configFlags.KubeConfig = &clientcmd.RecommendedHomeFile
	f := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(configFlags))
	do := f.NewBuilder().
		Unstructured().
		NamespaceParam("test").DefaultNamespace().AllNamespaces(false).
		ResourceTypeOrNameArgs(true, "deployment/productpage").
		ContinueOnError().
		Latest().
		Flatten().
		TransformRequests(func(req *rest.Request) { req.Param("includeObject", "Object") }).
		Do()
	if err := do.Err(); err != nil {
		log.Warn(err)
	}
	infos, err := do.Infos()
	if err != nil {
		log.Println(err)
	}
	for _, info := range infos {
		printer, err := printers.NewJSONPathPrinter("{.spec.selector}")
		if err != nil {
			log.Println(err)
		}
		buf := bytes.NewBuffer([]byte{})
		err = printer.PrintObj(info.Object, buf)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(buf.String())
		l := &metav1.LabelSelector{}
		err = json2.Unmarshal([]byte(buf.String()), l)
		if err != nil || len(l.MatchLabels) == 0 {
			m := map[string]string{}
			_ = json2.Unmarshal([]byte(buf.String()), &m)
			l = &metav1.LabelSelector{MatchLabels: m}
		}
		fmt.Println(l)
	}
	printer, err := printers.NewJSONPathPrinter("{.spec.template.spec.containers[0].ports}")
	portPrinter, err := printers.NewJSONPathPrinter("{.spec.ports}")
	var result []corev1.ContainerPort
	for _, info := range infos {
		buf := bytes.NewBuffer([]byte{})
		err = printer.PrintObj(info.Object, buf)
		if err != nil {
			_ = portPrinter.PrintObj(info.Object, buf)
			var ports []corev1.ServicePort
			_ = json2.Unmarshal([]byte(buf.String()), &ports)
			for _, port := range ports {
				val := port.TargetPort.IntVal
				if val == 0 {
					val = port.Port
				}
				result = append(result, corev1.ContainerPort{
					Name:          port.Name,
					ContainerPort: val,
					Protocol:      port.Protocol,
				})
			}
		} else {
			_ = json2.Unmarshal([]byte(buf.String()), &result)
		}
		fmt.Println(result)
	}
}

func TestGetTopController(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	configFlags.KubeConfig = &clientcmd.RecommendedHomeFile
	factory := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(configFlags))
	clientset, _ := factory.KubernetesClientSet()
	controller, err := util.GetTopControllerBaseOnPodLabel(
		factory,
		clientset.CoreV1().Pods("default"),
		"default",
		labels.SelectorFromSet(map[string]string{"": ""}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(controller)
	fmt.Println(controller.Name)
}

func TestUDP(t *testing.T) {
	go func() {
		server()
	}()
	time.Sleep(time.Second * 1)
	client()
}

func client() {
	socket, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		IP:   net.IPv4(172, 20, 225, 47),
		Port: 55555,
	})
	if err != nil {
		fmt.Println("连接失败!", err)
		return
	}
	defer socket.Close()

	// 发送数据
	senddata := []byte("hello server!")
	_, err = socket.Write(senddata)
	if err != nil {
		fmt.Println("发送数据失败!", err)
		return
	}

	// 接收数据
	data := make([]byte, 4096)
	read, remoteAddr, err := socket.ReadFromUDP(data)
	if err != nil {
		fmt.Println("读取数据失败!", err)
		return
	}
	fmt.Println(read, remoteAddr)
	fmt.Printf("%s\n", data[0:read])
}

func server() {
	// 创建监听
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 55555,
	})
	if err != nil {
		return
	}
	defer socket.Close()

	for {
		data := make([]byte, 4096)
		read, remoteAddr, err := socket.ReadFromUDP(data)
		if err != nil {
			fmt.Println("读取数据失败!", err)
			continue
		}
		fmt.Println(read, remoteAddr)
		fmt.Printf("%s\n\n", data[0:read])

		senddata := []byte("hello client!")
		_, err = socket.WriteToUDP(senddata, remoteAddr)
		if err != nil {
			fmt.Println("发送数据失败!", err)
			return
		}
	}
}

func TestGetMac(t *testing.T) {
	address := util.GetMacAddress()
	fmt.Println(address.String())
}

func TestPatchCm(t *testing.T) {
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: clientcmd.RecommendedHomeFile}, nil,
	)
	config, err := clientConfig.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	patch, err := clientset.CoreV1().ConfigMaps("default").Patch(
		context.Background(),
		"kubevpn.traffic.manager",
		types.MergePatchType,
		[]byte("{\"data\":{\"Connect\":\"1.1.1.1,2.2.2.666666\\nac:de:48:00:11:22\\n\"}}"),
		metav1.PatchOptions{},
	)
	fmt.Println(err)
	fmt.Println(patch)
}
