/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package setupcluster

import (
	"encoding/json"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/version"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"strconv"
)

type SetUpCluster interface {
	IsAdmin() *setUpCluster
	CreateNs(namespace, label string) *setUpCluster
	CreateConfigMap(name, namespace, key, value string) *setUpCluster
	DeployNocalhostDep(image, namespace string) *setUpCluster
	GetClusterNode() *setUpCluster
	GetClusterVersion() *setUpCluster
	GetClusterInfo() *setUpCluster
	GetErr() (string, error, error)
}

type setUpCluster struct {
	clientGo      *clientgo.GoClient
	err           error
	errCode       error
	nodeList      *corev1.NodeList
	serverVersion *version.Info
	clusterInfo   string
}

func NewSetUpCluster(client *clientgo.GoClient) SetUpCluster {
	return &setUpCluster{
		clientGo: client,
	}
}

func (c *setUpCluster) GetErr() (string, error, error) {
	return c.clusterInfo, c.err, c.errCode
}

func (c *setUpCluster) IsAdmin() *setUpCluster {
	_, c.err = c.clientGo.IsAdmin()
	if c.err != nil {
		c.errCode = errno.ErrClusterKubeAdmin
	}
	return c
}

func (c *setUpCluster) CreateNs(namespace, label string) *setUpCluster {
	_, _ = c.clientGo.CreateNS(namespace, label)
	return c
}

func (c *setUpCluster) CreateConfigMap(name, namespace, key, value string) *setUpCluster {
	_, c.err = c.clientGo.CreateConfigMap(name, namespace, key, value)
	if c.err != nil {
		c.errCode = errno.ErrClusterDepSetup
	}
	return c
}

func (c *setUpCluster) DeployNocalhostDep(image, namespace string) *setUpCluster {
	_, c.err = c.clientGo.DeployNocalhostDep(image, namespace)
	if c.err != nil {
		c.errCode = errno.ErrClusterDepJobSetup
	}
	return c
}

func (c *setUpCluster) GetClusterNode() *setUpCluster {
	nodeList, err := c.clientGo.GetClusterNode()
	if err != nil {
		c.err = err
	}
	c.nodeList = nodeList
	return c
}

func (c *setUpCluster) GetClusterVersion() *setUpCluster {
	version, err := c.clientGo.GetClusterVersion()
	if err != nil {
		c.err = err
	}
	c.serverVersion = version
	return c
}

func (c *setUpCluster) GetClusterInfo() *setUpCluster {
	info := map[string]interface{}{
		"cluster_version": c.serverVersion.GitVersion,
		"nodes":           strconv.Itoa(len(c.nodeList.Items)),
	}
	b, _ := json.Marshal(info)
	c.clusterInfo = string(b)
	return c
}
