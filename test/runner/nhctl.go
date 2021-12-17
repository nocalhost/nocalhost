/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package runner

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
	"math/rand"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func NewNhctl(namespace, kubeconfig, suitName string) *CLI {
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
		suitName:   suitName,
	}
	n, err := exec.LookPath("nhctl")
	if err != nil {
		c.cmd = filepath.Join(homedir.HomeDir(), ".nh", "bin", "nhctl")
	} else {
		c.cmd = n
	}
	return NewCLI(c, namespace)
}

func NewKubectl(namespace, kubeconfig, suitName string) *CLI {
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
		cmd:        "kubectl",
		suitName:   suitName,
	}
	return NewCLI(c, namespace)
}

func NewHelm(namespace, kubeconfig, suitName string) *CLI {
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
		cmd:        "helm",
		suitName:   suitName,
	}
	return NewCLI(c, namespace)
}

type Conf struct {
	kubeconfig string
	namespace  string
	cmd        string
	suitName   string
}

func (c *Conf) GetKubeConfig() string {
	return c.kubeconfig
}
func (c *Conf) GetNamespace() string {
	return c.namespace
}
func (c Conf) GetCmd() string {
	return c.cmd
}
func (c Conf) SuitName() string {
	return c.suitName
}

type Client interface {
	GetNhctl() *CLI
	GetKubectl() *CLI
	GetHelm() *CLI
	GetClientset() *kubernetes.Clientset

	NameSpace() string
	RandomNsCli(suitName string) Client
	SuiteName() string
}

func NewClient(kubeconfig, namespace, suitName string) Client {
	temp, _ := clientgoutils.NewClientGoUtils(kubeconfig, namespace)
	return &ClientImpl{
		kubeconfig: kubeconfig,
		namespace:  namespace,

		Nhctl:     NewNhctl(namespace, kubeconfig, suitName),
		Kubectl:   NewKubectl(namespace, kubeconfig, suitName),
		Helm:      NewHelm(namespace, kubeconfig, suitName),
		Clientset: temp.ClientSet,

		suitName: suitName,
	}
}

type ClientImpl struct {
	kubeconfig string
	namespace  string

	Nhctl     *CLI
	Kubectl   *CLI
	Helm      *CLI
	Clientset *kubernetes.Clientset

	suitName string
}

func (i *ClientImpl) SuiteName() string {
	return i.suitName
}

func (i *ClientImpl) NameSpace() string {
	return i.namespace
}

func (i *ClientImpl) RandomNsCli(suitName string) Client {
	ns := RandStringRunes()
	return NewClient(i.kubeconfig, ns, suitName)
}

func (i *ClientImpl) GetNhctl() *CLI {
	return i.Nhctl
}
func (i *ClientImpl) GetKubectl() *CLI {
	return i.Kubectl
}
func (i *ClientImpl) GetClientset() *kubernetes.Clientset {
	return i.Clientset
}
func (i *ClientImpl) GetHelm() *CLI {
	return i.Helm
}

var lock = sync.Mutex{}

func RandStringRunes() string {
	lock.Lock()
	defer lock.Unlock()

	// prevent seed conflict
	time.Sleep(100 * time.Millisecond)

	rand.Seed(time.Now().UnixNano())
	b := make([]rune, 10)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
