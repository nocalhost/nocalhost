/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package setupcluster

import (
	corev1 "k8s.io/api/core/v1"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

type ClusterDevsSetUp interface {
	CreateNS(namespace string, labels map[string]string) *clusterDevsSetUp
	CreateServiceAccount(name, namespace string) *clusterDevsSetUp
	CreateRole(name, namespace string) *clusterDevsSetUp
	CreateRoleBinding(name, namespace, clusterRole, toServiceAccount string) *clusterDevsSetUp
	GetServiceAccountSecret(name, namespace string) (*corev1.Secret, error)
	CreateResourceQuota(
		name, namespace, reqMem, reqCpu, limitsMem, limitsCpu, storageCapacity,
		ephemeralStorage, pvcCount, lbCount string,
	) *clusterDevsSetUp
	DeleteResourceQuota(name, namespace string) *clusterDevsSetUp
	CreateLimitRange(name, namespace, reqMem, limitsMem, reqCpu, limitsCpu, ephemeralStorage string) *clusterDevsSetUp
	DeleteLimitRange(name, namespace string) *clusterDevsSetUp
}

type clusterDevsSetUp struct {
	clientGo                 *clientgo.GoClient
	err                      error
	errCode                  error
	serviceAccountSecretName string
}

func (c *clusterDevsSetUp) CreateNS(namespace string, labels map[string]string) *clusterDevsSetUp {
	_, err := c.clientGo.CreateNS(namespace, labels)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindNameSpaceCreate
	}
	return c
}

func (c *clusterDevsSetUp) CreateServiceAccount(name, namespace string) *clusterDevsSetUp {
	_, err := c.clientGo.CreateServiceAccount(name, namespace)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindServiceAccountCreateErr
	}
	return c
}

func (c *clusterDevsSetUp) CreateRole(name, namespace string) *clusterDevsSetUp {
	_, err := c.clientGo.CreateRole(name, namespace)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindRoleCreateErr
	}
	return c
}

func (c *clusterDevsSetUp) CreateRoleBinding(name, namespace, role, toServiceAccount string) *clusterDevsSetUp {
	_, err := c.clientGo.CreateRoleBinding(name, namespace, role, toServiceAccount)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindRoleBindingCreateErr
	}
	return c
}

func (c *clusterDevsSetUp) CreateClusterRoleBinding(name, namespace, role, toServiceAccount string) *clusterDevsSetUp {
	_, err := c.clientGo.CreateClusterRoleBinding(name, namespace, role, toServiceAccount)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindRoleBindingCreateErr
	}
	return c
}

func (c *clusterDevsSetUp) GetServiceAccount(name, namespace string) *clusterDevsSetUp {
	serviceAccount, err := c.clientGo.WatchServiceAccount(name, namespace)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindSecretNameGetErr
	}
	if serviceAccount != nil && len(serviceAccount.Secrets) > 0 {
		c.serviceAccountSecretName = serviceAccount.Secrets[0].Name
	}
	return c
}

func (c *clusterDevsSetUp) GetServiceAccountSecret(name, namespace string) (*corev1.Secret, error) {
	if name == "" {
		name = c.serviceAccountSecretName
	}
	secret, err := c.clientGo.GetSecret(namespace, name)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrBindSecretGetErr
	}
	return secret, err
}

func (c *clusterDevsSetUp) CreateResourceQuota(
	name, namespace, reqMem, reqCpu, limitsMem,
	limitsCpu, storageCapacity, ephemeralStorage, pvcCount, lbCount string,
) *clusterDevsSetUp {
	_, err := c.clientGo.CreateResourceQuota(
		name, namespace, reqMem, reqCpu, limitsMem, limitsCpu, storageCapacity, ephemeralStorage, pvcCount, lbCount,
	)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrCreateResourceQuota
	}
	return c
}

func (c *clusterDevsSetUp) DeleteResourceQuota(name, namespace string) *clusterDevsSetUp {
	_, err := c.clientGo.DeleteResourceQuota(name, namespace)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrDeleteResourceQuota
	}
	return c
}

func (c *clusterDevsSetUp) CreateLimitRange(
	name, namespace, reqMem,
	limitsMem, reqCpu, limitsCpu, ephemeralStorage string,
) *clusterDevsSetUp {
	_, err := c.clientGo.CreateLimitRange(
		name, namespace, reqMem,
		limitsMem, reqCpu, limitsCpu, ephemeralStorage,
	)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrCreateLimitRange
	}
	return c
}

func (c *clusterDevsSetUp) DeleteLimitRange(name, namespace string) *clusterDevsSetUp {
	_, err := c.clientGo.DeleteLimitRange(name, namespace)
	if err != nil {
		c.err = err
		c.errCode = errno.ErrDeleteLimitRange
	}
	return c
}

func NewClusterDevsSetUp(c *clientgo.GoClient) ClusterDevsSetUp {
	return &clusterDevsSetUp{
		clientGo: c,
	}
}
