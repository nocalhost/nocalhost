/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runner

import (
	"k8s.io/client-go/kubernetes"
	"math/rand"
	"nocalhost/pkg/nhctl/clientgoutils"
	"os/exec"
	"sync"
	"time"
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func NewNhctl(namespace, kubeconfig string) *CLI {
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
	}
	n, err := exec.LookPath("nhctl")
	if err != nil {
		c.cmd = "nhctl"
	} else {
		c.cmd = n
	}
	return NewCLI(c, namespace)
}

func NewKubectl(namespace, kubeconfig string) *CLI {
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
		cmd:        "kubectl",
	}
	return NewCLI(c, namespace)
}

func NewHelm(namespace, kubeconfig string) *CLI {
	c := &Conf{
		kubeconfig: kubeconfig,
		namespace:  namespace,
		cmd:        "helm",
	}
	return NewCLI(c, namespace)
}

type Conf struct {
	kubeconfig string
	namespace  string
	cmd        string
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

type Client interface {
	GetNhctl() *CLI
	GetKubectl() *CLI
	GetHelm() *CLI
	GetClientset() *kubernetes.Clientset

	NameSpace() string
	RandomNsCli() Client
}

func NewClient(kubeconfig, namespace string) Client {
	temp, _ := clientgoutils.NewClientGoUtils(kubeconfig, namespace)
	return &ClientImpl{
		kubeconfig: kubeconfig,
		namespace:  namespace,

		Nhctl:     NewNhctl(namespace, kubeconfig),
		Kubectl:   NewKubectl(namespace, kubeconfig),
		Helm:      NewHelm(namespace, kubeconfig),
		Clientset: temp.ClientSet,
	}
}

type ClientImpl struct {
	kubeconfig string
	namespace  string

	Nhctl     *CLI
	Kubectl   *CLI
	Helm      *CLI
	Clientset *kubernetes.Clientset
}

func (i *ClientImpl) NameSpace() string {
	return i.namespace
}

func (i *ClientImpl) RandomNsCli() Client {
	ns := RandStringRunes()
	return NewClient(i.kubeconfig, ns)
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
