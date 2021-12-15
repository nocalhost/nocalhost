/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"bufio"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/cmd/util"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
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
	join := filepath.Join(clientcmd.RecommendedConfigDir, "config_backup")
	configFlags.KubeConfig = &join
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

	//out, err := Shell(clientset, restclient, config, TrafficManager, namespace, "cat /etc/resolv.conf | grep nameserver | awk '{print$2}'")
	//serviceList, err := clientset.CoreV1().Services(v1.NamespaceSystem).List(context.Background(), v1.ListOptions{
	//	FieldSelector: fields.OneTermEqualSelector("metadata.name", "kube-dns").String(),
	//})
	//
	//fmt.Println(out == serviceList.Items[0].Spec.ClusterIP)
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

func TestPatchAnnotation(t *testing.T) {
	TestShell(t)
	err := MergeOrReplaceAnnotation(
		clientset.CoreV1().RESTClient(),
		namespace,
		"pods",
		"productpage-shadow",
		"backup",
		"testing",
	)
	fmt.Println(err)
}

func TestLogger(t *testing.T) {
	reader, writer := io.Pipe()
	cancel, _ := context.WithCancel(context.WithValue(context.TODO(), "logger", &log.Logger{
		Out:          writer,
		Formatter:    new(log.TextFormatter),
		Hooks:        make(log.LevelHooks),
		Level:        log.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}))
	dial, _ := net.Dial("", "")
	go func() {
		newReader := bufio.NewReader(reader)
		for {
			line, _, _ := newReader.ReadLine()
			fmt.Println(string(line))
			io.WriteString(dial, string(line))
		}
	}()
	cancel.Value("logger").(*log.Logger).Infoln("this is a test1")
	cancel.Value("logger").(*log.Logger).Infoln("this is a test2")
	cancel.Value("logger").(*log.Logger).Infoln("this is a test3")

	time.Sleep(time.Second * 5)
}

func TestExecCommand(t *testing.T) {
	var err error

	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	join := filepath.Join(clientcmd.RecommendedConfigDir, "config")
	configFlags.KubeConfig = &join
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

	errors := make(chan error, 2)
	reader, writer := io.Pipe()
	go func() {
		if err = ShellWithStream(
			clientset,
			restclient,
			config,
			"deployments-details-shadow",
			namespace,
			"ping 223.254.254.2",
			writer,
		); err != nil {
			errors <- err
		}
	}()
	newReader := bufio.NewReader(reader)
	//icmp_seq=5 ttl=64 time=55.8
	compile, _ := regexp.Compile("icmp_seq=(.*?) ttl=(.*?) time=(.*?)")

	go func() {
		for {
			line, _, err := newReader.ReadLine()
			if err != nil {
				errors <- err
				return
			}
			fmt.Printf("%s --> %v\n", string(line), compile.MatchString(string(line)))
		}
	}()
	<-errors
	fmt.Println("failed")
}

func TestPing(t *testing.T) {
	var cmd *exec.Cmd
	if IsWindows() {
		cmd = exec.Command("ping", "223.254.254.100")
	} else {
		cmd = exec.Command("ping", "-c", "4", "223.254.254.100")
	}
	compile, _ := regexp.Compile("icmp_seq=(.*?) ttl=(.*?) time=(.*?)")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(false)
		return
	}
	if compile.MatchString(string(output)) {
		fmt.Println(true)
		return
	}
	fmt.Println(string(output))
}

func BenchmarkName(b *testing.B) {
	b.N = 10000
	for i := 0; i < b.N; i++ {
		TestPing(nil)
	}
}

func TestGetUnstructuredObject(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true).WithDeprecatedPasswordFlag()
	join := filepath.Join(clientcmd.RecommendedConfigDir, "mesh")
	configFlags.KubeConfig = &join
	factory := cmdutil.NewFactory(cmdutil.NewMatchVersionFlags(configFlags))
	_, err := GetUnstructuredObject(factory, "naison", "pods/details-77d4b77fcb-wcnz7")
	if err != nil {
		log.Fatal(err)
	}
	s := labels.SelectorFromSet(map[string]string{"app": "productpage"})
	fmt.Println(s.String())
	var va []ResourceTupleWithScale
	_, err = GetAndConsumeControllerObject(factory, "naison", s, func(u *unstructured.Unstructured) {
		replicas, _, err := unstructured.NestedInt64(u.Object, "spec", "replicas")
		if err != nil {
			return
		}
		va = append(va, ResourceTupleWithScale{
			Resource: strings.ToLower(u.GetKind()) + "s",
			Name:     u.GetName(),
			Scale:    int(replicas),
		})
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(va)
}
