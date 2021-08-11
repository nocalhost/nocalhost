/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package service_account

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"sort"

	"sync"
)

// the user that has all verbs with all cluster resources
const CLUSTER_ADMIN PrivilegeType = "CLUSTER_ADMIN"

// the user that has (get, list, watch) verbs with all cluster resources
const CLUSTER_VIEWER PrivilegeType = "CLUSTER_VIEWER"

// user do not has cluster resources access permissions
const NONE PrivilegeType = "NONE"

type PrivilegeType string

type SaAuthorizeRequest struct {
	ClusterId *uint64 `json:"cluster_id" binding:"required"`
	UserId    *uint64 `json:"user_id" binding:"required"`
	SpaceName string  `json:"space_name" binding:"required"`
}

func Authorize(c *gin.Context) {
	var req SaAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind service account authorizeRequest params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}

	err := service.Svc.AuthorizeNsToUser(*req.ClusterId, *req.UserId, req.SpaceName)
	if err != nil {
		api.SendResponse(c, err, nil)
		return
	}

	api.SendResponse(c, nil, nil)
}

func ListAuthorization(c *gin.Context) {
	userId, err := ginbase.LoginUser(c)
	if err != nil {
		api.SendResponse(c, errno.ErrLoginRequired, nil)
		return
	}

	// optimization required
	clusters, err := service.Svc.ClusterSvc().GetList(c)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	user, err := service.Svc.UserSvc().GetCache(userId)
	if err != nil {
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	result := []*ServiceAccountModel{}
	var lock sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(clusters))

	for _, cluster := range clusters {
		cluster := cluster
		go func() {
			defer wg.Done()
			GenKubeconfig(
				user.SaName, cluster, "",
				func(nss []NS, privilegeType PrivilegeType, kubeConfig string) {
					if len(nss) != 0 || privilegeType != NONE {
						sort.Slice(
							nss, func(i, j int) bool {
								return nss[i].Namespace > nss[j].Namespace
							},
						)

						lock.Lock()
						result = append(
							result, &ServiceAccountModel{
								ClusterId:     cluster.ID,
								KubeConfig:    kubeConfig,
								StorageClass:  cluster.StorageClass,
								NS:            nss,
								Privilege:     privilegeType != NONE,
								PrivilegeType: privilegeType,
							},
						)
						lock.Unlock()
					}
				},
			)
		}()
	}

	sort.Slice(
		result, func(i, j int) bool {
			return result[i].ClusterId > result[j].ClusterId
		},
	)

	wg.Wait()
	api.SendResponse(c, nil, result)
}

func GenKubeconfig(
	saName string, cp model.ClusterPack,
	specifyNameSpace string,
	then func(nss []NS, privilegeType PrivilegeType, kubeConfig string),
) {
	// new client go
	clientGo, err := clientgo.NewAdminGoClient([]byte(cp.GetKubeConfig()))
	if err != nil {
		return
	}

	// nocalhost provide every user a service account each cluster
	// first check if config valid
	var reader setupcluster.DevKubeConfigReader
	if reader = getServiceAccountKubeConfigReader(
		clientGo, saName,
		_const.NocalhostDefaultSaNs, cp.GetClusterServer(),
	); reader == nil {
		return
	}

	var kubeConfig string
	if kubeConfig, _, _ = reader.ToYamlString(); kubeConfig == "" {
		return
	}

	// rewrite context info if needed
	// for un privilege cluster, should append all devspace to it's context
	kubeConfigStruct, _, _ := reader.ToStruct()
	kubeConfigStruct.Clusters[0].Name = cp.GetClusterName()
	kubeConfigStruct.Contexts[0].Context.Cluster = cp.GetClusterName()
	authInfo := kubeConfigStruct.Contexts[0].Context.AuthInfo

	kubeConfigStruct.Contexts = []clientcmdapiv1.NamedContext{}

	// then check if has privilege (cluster admin)
	privilegeType := NONE
	var nss []NS

	// new admin go client will request authorizationv1.SelfSubjectAccessReview
	// then did not find any err, means current user is the cluster admin role
	// todo we should specify the
	if cluster_scope.IsOwnerOfCluster(cp.GetClusterId(), saName) ||
		cluster_scope.IsCooperOfCluster(cp.GetClusterId(), saName) {
		privilegeType = CLUSTER_ADMIN
	} else if cluster_scope.IsViewerOfCluster(cp.GetClusterId(), saName) {
		privilegeType = CLUSTER_VIEWER
	}

	// different kind of namespace's permission with different prefix
	doForNamespaces := func(namespaces []string, spaceOwnType model.SpaceOwnType) {
		for _, ns := range namespaces {

			cu, err := service.Svc.ClusterUser().
				GetCacheByClusterAndNameSpace(cp.GetClusterId(), ns)

			if err != nil {
				log.Error(
					errors.Wrap(
						err, fmt.Sprintf(
							"Error while gen kubeconfig, can not find "+
								"devspace by clusterId %v and namespace %v", cp.GetClusterId(), ns,
						),
					),
				)
			} else {
				kubeConfigStruct.Contexts = append(
					kubeConfigStruct.Contexts, clientcmdapiv1.NamedContext{
						Name: cu.SpaceName,
						Context: clientcmdapiv1.Context{
							Namespace: ns,
							Cluster:   cp.GetClusterName(),
							AuthInfo:  authInfo,
						},
					},
				)

				nss = append(
					nss, NS{SpaceName: cu.SpaceName, Namespace: ns, SpaceId: cu.ID, SpaceOwnType: spaceOwnType.Str},
				)
			}
		}
	}

	doForNamespaces(ns_scope.GetAllOwnNs(string(clientGo.Config), saName), model.DevSpaceOwnTypeOwner)
	doForNamespaces(ns_scope.GetAllCoopNs(string(clientGo.Config), saName), model.DevSpaceOwnTypeCooperator)
	doForNamespaces(ns_scope.GetAllViewNs(string(clientGo.Config), saName), model.DevSpaceOwnTypeViewer)

	if len(nss) > 0 {
		// sort nss
		sort.Slice(
			nss, func(i, j int) bool {
				if nss[i].Namespace == specifyNameSpace {
					return false
				}
				if nss[j].Namespace == specifyNameSpace {
					return true
				}

				return nss[i].SpaceId > nss[i].SpaceId
			},
		)

		sort.Slice(
			kubeConfigStruct.Contexts, func(i, j int) bool {
				if nss[i].Namespace == specifyNameSpace {
					return false
				}
				if nss[j].Namespace == specifyNameSpace {
					return true
				}

				return nss[i].SpaceId > nss[i].SpaceId
			},
		)
		kubeConfigStruct.CurrentContext = nss[len(nss)-1].SpaceName
	} else {

		// while user has cluster - scope permission
		// and without any namespace - scope custom permission
		// we should add a default context for him
		if privilegeType != NONE {
			kubeConfigStruct.Contexts = append(
				kubeConfigStruct.Contexts, clientcmdapiv1.NamedContext{
					Name: "default",
					Context: clientcmdapiv1.Context{
						Namespace: "default",
						Cluster:   cp.GetClusterName(),
						AuthInfo:  authInfo,
					},
				},
			)

			kubeConfigStruct.CurrentContext = "default"
		}
	}

	if kubeConfig, _, _ = reader.ToYamlString(); kubeConfig == "" {
		return
	}
	then(nss, privilegeType, kubeConfig)
}

func GetCluster2Ns2SpaceNameMapping(
	devSpaces []*model.ClusterUserModel,
) map[uint64]map[string]*model.ClusterUserModel {
	spaceNameMap := map[uint64]map[string]*model.ClusterUserModel{}
	for _, space := range devSpaces {
		m, ok := spaceNameMap[space.ClusterId]
		if !ok {
			m = map[string]*model.ClusterUserModel{}
		}

		m[space.Namespace] = space
		spaceNameMap[space.ClusterId] = m
	}
	return spaceNameMap
}

func getServiceAccountKubeConfigReader(
	clientGo *clientgo.GoClient,
	saName, saNs, serverAddr string,
) setupcluster.DevKubeConfigReader {
	sa, err := clientGo.GetServiceAccount(saName, saNs)
	if err != nil || len(sa.Secrets) == 0 {
		return nil
	}

	secret, err := clientGo.GetSecret(_const.NocalhostDefaultSaNs, sa.Secrets[0].Name)
	if err != nil {
		return nil
	}
	cr := setupcluster.NewDevKubeConfigReader(
		secret, serverAddr, saNs,
	)

	cr.GetCA().GetToken().AssembleDevKubeConfig()
	return cr
}

type ServiceAccountModel struct {
	ClusterId     uint64        `json:"cluster_id"`
	KubeConfig    string        `json:"kubeconfig"`
	StorageClass  string        `json:"storage_class"`
	NS            []NS          `json:"namespace_packs"`
	Privilege     bool          `json:"privilege"`
	PrivilegeType PrivilegeType `json:"privilege_type"`
}

type NS struct {
	SpaceId      uint64 `json:"space_id"`
	Namespace    string `json:"namespace"`
	SpaceName    string `json:"spacename"`
	SpaceOwnType string `json:"space_own_type"`
}
