package setupcluster

import (
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

type SetUpCluster interface {
	IsAdmin() *setUpCluster
	CreateNs(namespace, label string) *setUpCluster
	CreateConfigMap(name, namespace, key, value string) *setUpCluster
	DeployNocalhostDep(image, namespace string) *setUpCluster
	GetErr() (error, error)
}

type setUpCluster struct {
	clientGo *clientgo.GoClient
	err      error
	errCode  error
}

func NewSetUpCluster(client *clientgo.GoClient) SetUpCluster {
	return &setUpCluster{
		clientGo: client,
	}
}

func (c *setUpCluster) GetErr() (error, error) {
	return c.err, c.errCode
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
