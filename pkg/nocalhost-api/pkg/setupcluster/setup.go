/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package setupcluster

import (
	"encoding/json"
	"errors"
	apiappsV1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/version"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strconv"
	"strings"
)

type SetUpCluster interface {
	IsAdmin() (bool, error)
	CreateNs(namespace string, labels map[string]string) *setUpCluster
	CreateConfigMap(name, namespace, key, value string) *setUpCluster
	DeployNocalhostDep(namespace, serviceAccount, tag string) *setUpCluster
	GetClusterNode() *setUpCluster
	GetClusterVersion() *setUpCluster
	GetClusterInfo() *setUpCluster
	CreateServiceAccount(name, namespace string) *setUpCluster
	CreateClusterRoleBinding(name, namespace, role, toServiceAccount string) *setUpCluster
	CreateNocalhostPriorityClass() *setUpCluster
	GetErr() (string, error, error)
	InitCluster(tag string) (string, error, error)
	UpgradeCluster() (bool, error)
}

type setUpCluster struct {
	clientGo      *clientgo.GoClient
	err           error
	errCode       error
	nodeList      *corev1.NodeList
	serverVersion *version.Info
	clusterInfo   string
}

// NewSetUpCluster cluster set up is a simple builder to
// operate the cluster, if err occur, we should store the err msg
// in err and errCode and then take it out at the end
// then we can skip same error while operate the k8s cluster
func NewSetUpCluster(client *clientgo.GoClient) SetUpCluster {
	return &setUpCluster{
		clientGo: client,
	}
}

func (c *setUpCluster) GetErr() (string, error, error) {
	return c.clusterInfo, c.err, c.errCode
}

func (c *setUpCluster) IsAdmin() (bool, error) {
	return c.clientGo.IsAdmin()
}

func (c *setUpCluster) CreateNs(namespace string, labels map[string]string) *setUpCluster {
	_, _ = c.clientGo.CreateNS(namespace, labels)
	return c
}

func (c *setUpCluster) CreateServiceAccount(name, namespace string) *setUpCluster {
	_, err := c.clientGo.CreateServiceAccount(name, namespace)
	if err != nil {
		c.errCode = errno.ErrBindServiceAccountCreateErr
	}
	return c
}

func (c *setUpCluster) CreateClusterRoleBinding(name, namespace, role, toServiceAccount string) *setUpCluster {
	_, err := c.clientGo.CreateClusterRoleBinding(name, namespace, role, toServiceAccount)
	if err != nil {
		c.errCode = errno.ErrBindRoleBindingCreateErr
	}
	return c
}

func (c *setUpCluster) CreateConfigMap(name, namespace, key, value string) *setUpCluster {
	_, c.err = c.clientGo.CreateConfigMapWithValue(name, namespace, key, value)
	if c.err != nil {
		c.errCode = errno.ErrClusterDepSetup
	}
	return c
}

func (c *setUpCluster) DeployNocalhostDep(namespace, serviceAccount, tag string) *setUpCluster {
	_, c.err = c.clientGo.DeployNocalhostDep(namespace, serviceAccount, tag)
	if c.err != nil {
		c.errCode = errno.ErrClusterDepJobSetup
	}
	return c
}

func (c *setUpCluster) CreateNocalhostPriorityClass() *setUpCluster {
	c.err = c.clientGo.CreateNocalhostPriorityClass()
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
	cVersion, err := c.clientGo.GetClusterVersion()
	if err != nil {
		c.err = err
	}
	c.serverVersion = cVersion
	return c
}

func (c *setUpCluster) GetClusterInfo() *setUpCluster {
	if c.err != nil {
		return c
	}
	info := map[string]interface{}{
		"cluster_version": c.serverVersion.GitVersion,
		"nodes":           strconv.Itoa(len(c.nodeList.Items)),
	}
	b, _ := json.Marshal(info)
	c.clusterInfo = string(b)
	return c
}

func (c *setUpCluster) InitCluster(tag string) (string, error, error) {
	return c.CreateNs(global.NocalhostSystemNamespace, map[string]string{}).
		CreateServiceAccount(global.NocalhostSystemNamespaceServiceAccount, global.NocalhostSystemNamespace).
		CreateClusterRoleBinding(
			global.NocalhostSystemRoleBindingName, global.NocalhostSystemNamespace, "cluster-admin",
			global.NocalhostSystemNamespaceServiceAccount,
		).
		CreateNocalhostPriorityClass().
		DeployNocalhostDep(global.NocalhostSystemNamespace, global.NocalhostSystemNamespaceServiceAccount, tag).
		GetClusterNode().
		GetClusterVersion().
		GetClusterInfo().
		GetErr()
}

func (c *setUpCluster) UpgradeCluster() (bool, error) {
	existPc, _ := c.clientGo.ExistPriorityClass(global.NocalhostDefaultPriorityclassName)
	if !existPc {

		log.Info("PriorityClass " + global.NocalhostDefaultPriorityclassName + " is not exist so creat one.")
		c.CreateNocalhostPriorityClass()

		if c.err != nil {
			return false, c.err
		}
	}

	existNs, _ := c.clientGo.ExistNs(global.NocalhostSystemNamespace)
	if !existNs {

		log.Info("Namespace " + global.NocalhostSystemNamespace + " is not exist so creat one.")
		c.CreateNs(global.NocalhostSystemNamespace, map[string]string{})

		if c.err != nil {
			return false, c.err
		}
	}

	existServiceAccount, _ := c.clientGo.ExistServiceAccount(
		global.NocalhostSystemNamespace, global.NocalhostSystemNamespaceServiceAccount,
	)
	if !existServiceAccount {

		log.Info("ServiceAccount " + global.NocalhostSystemNamespaceServiceAccount + " is not exist so creat one.")
		c.CreateServiceAccount(global.NocalhostSystemNamespaceServiceAccount, global.NocalhostSystemNamespace)

		if c.err != nil {
			return false, c.err
		}
	}

	existClusterRoleBinding, _ := c.clientGo.ExistClusterRoleBinding(global.NocalhostSystemRoleBindingName)
	if !existClusterRoleBinding {

		log.Info("ClusterAdmin-RoleBinding " + global.NocalhostSystemRoleBindingName + " is not exist so creat one.")
		c.CreateClusterRoleBinding(
			global.NocalhostSystemRoleBindingName, global.NocalhostSystemNamespace, "cluster-admin",
			global.NocalhostSystemNamespaceServiceAccount,
		)

		if c.err != nil {
			return false, c.err
		}
	}

	existDeployment, deployment := c.clientGo.ExistDeployment(global.NocalhostSystemNamespace, global.NocalhostDepName)
	if !existDeployment || !c.CheckIfSameImage(deployment, c.clientGo.MatchedArtifactVersion(clientgo.Dep, "")) {

		log.Info("Re-deploying nocalhost-dep... ")
		c.DeleteOldDepJob(global.NocalhostSystemNamespace)
		c.DeployNocalhostDep(global.NocalhostSystemNamespace, global.NocalhostSystemNamespaceServiceAccount, "")

		if c.err != nil {
			return false, c.err
		}
	}

	return true, nil
}

func (c *setUpCluster) CheckIfSameImage(deployment *apiappsV1.Deployment, image string) (same bool) {
	containers := deployment.Spec.Template.Spec.Containers

	switch len(containers) {
	case 0:
		c.err = errors.New("None container in dep-deployment ")
		return
	case 1:
		break
	default:
		c.err = errors.New("Multi containers in dep-deployment ")
		return
	}

	if image != containers[0].Image {
		log.Infof("Current image " + containers[0].Image + " is different from version matched " + image)
		return
	} else {
		same = true
		return
	}
}

func (c *setUpCluster) DeleteOldDepJob(namespace string) {
	jobs, err := c.clientGo.ListJobs(namespace)
	if err == nil {
		for _, item := range jobs.Items {
			if strings.HasPrefix(item.Name, global.NocalhostDepJobNamePrefix) {
				_ = c.clientGo.DeleteJob(namespace, item.Name)
			}
		}
	}

	pods, err := c.clientGo.ListPods(namespace)
	if err == nil {
		for _, item := range pods.Items {
			if strings.HasPrefix(item.Name, global.NocalhostDepJobNamePrefix) {
				_ = c.clientGo.DeletePod(namespace, item.Name)
			}
		}
	}
}
