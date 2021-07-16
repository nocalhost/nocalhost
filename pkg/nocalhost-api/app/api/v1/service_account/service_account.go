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

package service_account

import (
	"context"
	"github.com/gin-gonic/gin"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"sort"

	"sync"
)

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

	user, err := service.Svc.UserSvc().GetUserByID(c, userId)
	if err != nil {
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	devSpaces, err := service.Svc.ClusterUser().GetList(context.TODO(), model.ClusterUserModel{})
	if err != nil {
		api.SendResponse(c, errno.ErrClusterUserNotFound, nil)
		return
	}

	spaceNameMap := GetCluster2Ns2SpaceNameMapping(devSpaces)

	result := []*ServiceAccountModel{}
	var lock sync.Mutex
	wg := sync.WaitGroup{}
	wg.Add(len(clusters))

	for _, cluster := range clusters {
		cluster := cluster
		go func() {
			defer wg.Done()
			GenKubeconfig(
				user.SaName, cluster, spaceNameMap, "",
				func(nss []NS, privilege bool, kubeConfig string) {
					if len(nss) != 0 || privilege {
						sort.Slice(
							nss, func(i, j int) bool {
								return nss[i].Namespace > nss[j].Namespace
							},
						)

						lock.Lock()
						result = append(
							result, &ServiceAccountModel{
								ClusterId:    cluster.ID,
								KubeConfig:   kubeConfig,
								StorageClass: cluster.StorageClass,
								NS:           nss,
								Privilege:    privilege,
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
	spaceNameMap map[uint64]map[string]*model.ClusterUserModel,
	specifyNameSpace string,
	then func(nss []NS, privilege bool, kubeConfig string),
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
		service.NocalhostDefaultSaNs, cp.GetClusterServer(),
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
	privilege := false
	var nss []NS

	// new admin go client will request authorizationv1.SelfSubjectAccessReview
	// then did not find any err, means cluster admin
	if _, err = clientgo.NewAdminGoClient([]byte(kubeConfig)); err == nil {

		privilege = true
		// if namespace provided, set the space to current context

		// if found the specified ns found, set the context to those space
		// else gen a empty space named default ant set to default
		found := false
		kubeConfigStruct.Contexts = []clientcmdapiv1.NamedContext{}

		if specifyNameSpace != "" && specifyNameSpace != "*" {
			if m, ok := spaceNameMap[cp.GetClusterId()]; ok {
				if s, ok := m[specifyNameSpace]; ok {
					kubeConfigStruct.Contexts = append(
						kubeConfigStruct.Contexts, clientcmdapiv1.NamedContext{
							Name: s.SpaceName,
							Context: clientcmdapiv1.Context{
								Namespace: specifyNameSpace,
								Cluster:   cp.GetClusterName(),
								AuthInfo:  authInfo,
							},
						},
					)

					kubeConfigStruct.CurrentContext = s.SpaceName
					found = true
				}
			}
		}

		if !found {
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

	} else {
		for _, ns := range GetAllPermittedNs(string(clientGo.Config), saName) {
			//var spaceName = fmt.Sprintf("Nocalhost-%s", ns)
			//var SpaceId = uint64(0)
			if m, ok := spaceNameMap[cp.GetClusterId()]; ok {
				if s, ok := m[ns]; ok {
					kubeConfigStruct.Contexts = append(
						kubeConfigStruct.Contexts, clientcmdapiv1.NamedContext{
							Name: s.SpaceName,
							Context: clientcmdapiv1.Context{
								Namespace: ns,
								Cluster:   cp.GetClusterName(),
								AuthInfo:  authInfo,
							},
						},
					)

					nss = append(
						nss, NS{SpaceName: s.SpaceName, Namespace: ns, SpaceId: s.ID},
					)
				}
			}
		}

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

			sort.Slice(kubeConfigStruct.Contexts, func(i, j int) bool {
				if nss[i].Namespace == specifyNameSpace {
					return false
				}
				if nss[j].Namespace == specifyNameSpace {
					return true
				}

				return nss[i].SpaceId > nss[i].SpaceId
			})
			kubeConfigStruct.CurrentContext = nss[len(nss)-1].SpaceName
		}
	}

	if kubeConfig, _, _ = reader.ToYamlString(); kubeConfig == "" {
		return
	}
	then(nss, privilege, kubeConfig)
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

	secret, err := clientGo.GetSecret(sa.Secrets[0].Name, service.NocalhostDefaultSaNs)
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
	ClusterId    uint64 `json:"cluster_id"`
	KubeConfig   string `json:"kubeconfig"`
	StorageClass string `json:"storage_class"`
	NS           []NS   `json:"namespace_packs"`
	Privilege    bool   `json:"privilege"`
}

type NS struct {
	SpaceId   uint64 `json:"space_id"`
	Namespace string `json:"namespace"`
	SpaceName string `json:"spacename"`
}
