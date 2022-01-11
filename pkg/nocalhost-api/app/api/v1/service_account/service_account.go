/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package service_account

import (
	"fmt"
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapiv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/manager"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
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

	result := make([]*ServiceAccountModel, 0, len(clusters))
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

	wg.Wait()

	devSpaces, _ := service.Svc.ClusterUser().GetList(c, model.ClusterUserModel{
		UserId:       userId,
		DevSpaceType: model.VirtualClusterType},
	)

	// add vcluster kubeconfig to result
	if len(devSpaces) != 0 {
		result = setVClusterKubeConfig(devSpaces, clusters, result)

		// remove namespace form vcluster devspace
		nsMap := make(map[string]struct{})
		for _, devSpace := range devSpaces {
			nsMap[devSpace.Namespace] = struct{}{}
		}
		for i := 0; i < len(result); i++ {
			ns := result[i].NS
			for j := 0; j < len(ns); j++ {
				if _, ok := nsMap[ns[j].Namespace]; !ok {
					continue
				}
				ns = ns[:j+copy(ns[j:], ns[j+1:])]
				j--
			}
			if len(ns) == 0 && !result[i].Privilege {
				result = result[:i+copy(result[i:], result[i+1:])]
				i--
				continue
			}
			result[i].NS = ns
		}
	}

	sort.Slice(
		result, func(i, j int) bool {
			if result[i].DevSpaceType != result[j].DevSpaceType {
				return result[i].DevSpaceType < result[j].DevSpaceType
			}
			return result[i].ClusterId > result[j].ClusterId
		},
	)

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
	hostConfig, err := clientcmd.Load([]byte(cp.GetKubeConfig()))
	if err != nil {
		log.Warnf("load host kubeconfig error: %v", err)
		return
	}

	hostCtx := hostConfig.Contexts[hostConfig.CurrentContext]
	if hostCtx == nil {
		log.Warn("host kubeconfig context not found")
		return
	}
	hostCluster := hostConfig.Clusters[hostCtx.Cluster]
	v1Cluster := &clientcmdapiv1.Cluster{}
	if err := clientcmdapiv1.Convert_api_Cluster_To_v1_Cluster(hostCluster, v1Cluster, nil); err != nil {
		log.Warnf("convert host kubeconfig error: %v", err)
		return
	}
	kubeConfigStruct.Clusters[0].Cluster = *v1Cluster
	kubeConfigStruct.Clusters[0].Name = cp.GetClusterName()
	kubeConfigStruct.Contexts[0].Context.Cluster = cp.GetClusterName()
	authInfo := kubeConfigStruct.Contexts[0].Context.AuthInfo

	kubeConfigStruct.Contexts = []clientcmdapiv1.NamedContext{}

	// then check if has privilege (cluster admin)
	privilegeType := NONE
	var nss []NS

	allDevSpace, err := service.Svc.ClusterUser().ListV2(model.ClusterUserModel{})
	devSpaceMapping := map[string]model.ClusterUserV2{}
	for _, cu := range allDevSpace {
		if !cu.IsClusterAdmin() {
			devSpaceMapping[cu.Namespace] = *cu
		}
	}

	// different kind of namespace's permission with different prefix
	doForNamespaces := func(namespaces []string, spaceOwnType model.SpaceOwnType) {
		for _, ns := range namespaces {
			cu, ok := devSpaceMapping[ns]

			if !ok {
				log.Error(
					errors.Errorf(
						"Error while gen kubeconfig, can not find "+
							"devspace by clusterId %v and namespace %v", cp.GetClusterId(), ns,
					),
				)
			} else {
				kubeConfigStruct.Contexts = append(
					kubeConfigStruct.Contexts, clientcmdapiv1.NamedContext{
						Name: fmt.Sprintf("%s/%s", cu.SpaceName, cu.Namespace),
						Context: clientcmdapiv1.Context{
							Namespace: ns,
							Cluster:   cp.GetClusterName(),
							AuthInfo:  authInfo,
						},
					},
				)

				nss = append(
					nss, NS{
						SpaceId:      cu.ID,
						Namespace:    ns,
						SpaceName:    cu.SpaceName,
						SleepStatus:  cu.SleepStatus,
						SpaceOwnType: spaceOwnType.Str,
					},
				)
				delete(devSpaceMapping, ns)
			}
		}
	}

	doForNamespaces(ns_scope.GetAllOwnNs(string(clientGo.Config), saName), model.DevSpaceOwnTypeOwner)
	doForNamespaces(ns_scope.GetAllCoopNs(string(clientGo.Config), saName), model.DevSpaceOwnTypeCooperator)
	doForNamespaces(ns_scope.GetAllViewNs(string(clientGo.Config), saName), model.DevSpaceOwnTypeViewer)

	remainNs := make([]string, 0)
	for _, cu := range devSpaceMapping {
		remainNs = append(remainNs, cu.Namespace)
	}

	// new admin go client will request authorizationv1.SelfSubjectAccessReview
	// then did not find any err, means current user is the cluster admin role
	// todo we should specify the
	if cluster_scope.IsOwnerOfCluster(cp.GetClusterId(), saName) {
		privilegeType = CLUSTER_ADMIN
		doForNamespaces(remainNs, model.DevSpaceOwnTypeOwner)
	} else if cluster_scope.IsCooperOfCluster(cp.GetClusterId(), saName) {
		privilegeType = CLUSTER_ADMIN
		doForNamespaces(remainNs, model.DevSpaceOwnTypeCooperator)
	} else if cluster_scope.IsViewerOfCluster(cp.GetClusterId(), saName) {
		privilegeType = CLUSTER_VIEWER
		doForNamespaces(remainNs, model.DevSpaceOwnTypeViewer)
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

				return nss[i].SpaceId > nss[j].SpaceId
			},
		)

		sort.Slice(
			kubeConfigStruct.Contexts, func(i, j int) bool {
				return kubeConfigStruct.Contexts[i].Name > kubeConfigStruct.Contexts[j].Name
			},
		)
		current := nss[len(nss)-1]
		kubeConfigStruct.CurrentContext = fmt.Sprintf("%s/%s", current.SpaceName, current.Namespace)
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

func GenVirtualClusterKubeConfig(clusterKubeConfig, spaceName, namespace string, f func(kubeConfig string, serviceType string)) {

	factory := manager.VClusterSharedManagerFactory
	vcManager, err := factory.Manager(clusterKubeConfig)
	if err != nil {
		f("", "")
		return
	}

	kubeConfig, serviceType, _ := vcManager.GetKubeConfig(spaceName, namespace)
	f(kubeConfig, serviceType)
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

func setVClusterKubeConfig(
	devSpaces []*model.ClusterUserModel, clusters []*model.ClusterList, result []*ServiceAccountModel) []*ServiceAccountModel {

	clustersMap := make(map[uint64]*model.ClusterList)
	for _, cluster := range clusters {
		clustersMap[cluster.ID] = cluster
	}

	accountMaps := make(map[uint64]*ServiceAccountModel)
	for i, accountModel := range result {
		accountMaps[accountModel.ClusterId] = result[i]
	}
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	for _, space := range devSpaces {
		clusterKubeConfig := clustersMap[space.ClusterId].GetKubeConfig()
		if clusterKubeConfig == "" {
			continue
		}

		space := space

		wg.Add(1)
		go func() {
			defer wg.Done()
			GenVirtualClusterKubeConfig(clusterKubeConfig, space.SpaceName, space.Namespace, func(
				kubeConfig, serviceType string) {
				if kubeConfig == "" {
					log.Error("get kube config failed")
					return
				}
				hostClusterContext, virtualClusterContext := "", ""
				if serviceType == string(corev1.ServiceTypeClusterIP) {
					var err error
					kubeConfig, hostClusterContext, virtualClusterContext, err = addHostContextIntoKubeConfig(kubeConfig, space, accountMaps)
					if err != nil {
						log.Error(err)
						return
					}
				}

				mu.Lock()
				result = append(result, &ServiceAccountModel{
					ClusterId:      space.ID + 1<<16,
					KubeConfig:     kubeConfig,
					StorageClass:   "",
					NS:             []NS{},
					Privilege:      true,
					PrivilegeType:  CLUSTER_ADMIN,
					DevSpaceType:   model.VirtualClusterType,
					KubeConfigType: "vcluster",
					VirtualCluster: VirtualCluster{
						ServiceType:           serviceType,
						ServicePort:           "443",
						ServiceAddress:        "service/" + global.VClusterPrefix + space.Namespace,
						ServiceNamespace:      space.Namespace,
						HostClusterContext:    hostClusterContext,
						VirtualClusterContext: virtualClusterContext,
					},
				})
				mu.Unlock()
			})
		}()
	}
	wg.Wait()
	return result
}

func addHostContextIntoKubeConfig(
	kubeConfig string, space *model.ClusterUserModel, accountMaps map[uint64]*ServiceAccountModel) (
	newKubeConfig, hostClusterContext, virtualClusterContext string, err error) {

	hostAccount := accountMaps[space.ClusterId]
	if hostAccount == nil {
		err = errors.Errorf("get host cluster kubeconfig failed, cluster id: %d", space.ClusterId)
		return
	}

	clusterKC, err := clientcmd.Load([]byte(hostAccount.KubeConfig))
	if err != nil {
		err = errors.Errorf("load host cluster kubeconfig failed, cluster id: %d, err: %v", space.ClusterId, err)
		return
	}

	hostClusterContext = clusterKC.CurrentContext
	hostCtx := clusterKC.Contexts[hostClusterContext]
	if hostCtx == nil {
		err = errors.Errorf("get host cluster kubeconfig context failed, cluster id: %d", space.ClusterId)
		return
	}

	hostCluster := clusterKC.Clusters[hostCtx.Cluster]
	if hostCluster == nil {
		err = errors.Errorf("get host cluster kubeconfig failed, cluster id: %d", space.ClusterId)
		return
	}
	hostAuth := clusterKC.AuthInfos[hostCtx.AuthInfo]
	if hostAuth == nil {
		err = errors.Errorf("get host cluster kubeconfig failed, cluster id: %d", space.ClusterId)
		return
	}

	kc, err := clientcmd.Load([]byte(kubeConfig))
	if err != nil {
		log.Errorf("load kubeconfig error: %s", err)
		return
	}
	virtualClusterContext = kc.CurrentContext
	kc.Contexts[hostClusterContext] = hostCtx
	kc.Clusters[hostCtx.Cluster] = hostCluster
	kc.AuthInfos[hostCtx.AuthInfo] = hostAuth

	kubeConfigByte, err := clientcmd.Write(*kc)
	if err != nil {
		log.Errorf("write kubeconfig error: %s", err)
		return
	}
	newKubeConfig = string(kubeConfigByte)
	return
}

type ServiceAccountModel struct {
	ClusterId      uint64         `json:"cluster_id"`
	KubeConfig     string         `json:"kubeconfig"`
	StorageClass   string         `json:"storage_class"`
	NS             []NS           `json:"namespace_packs"`
	Privilege      bool           `json:"privilege"`
	PrivilegeType  PrivilegeType  `json:"privilege_type"`
	DevSpaceType   uint64         `json:"dev_space_type"`
	KubeConfigType string         `json:"kubeconfig_type"`
	VirtualCluster VirtualCluster `json:"virtual_cluster"`
}

type NS struct {
	SpaceId      uint64 `json:"space_id"`
	Namespace    string `json:"namespace"`
	SpaceName    string `json:"spacename"`
	SleepStatus  string `json:"sleep_status"`
	SpaceOwnType string `json:"space_own_type"`
}

type VirtualCluster struct {
	ServiceType           string `json:"service_type"`
	ServicePort           string `json:"service_port"`
	ServiceAddress        string `json:"service_address"`
	ServiceNamespace      string `json:"service_namespace"`
	HostClusterContext    string `json:"host_cluster_context"`
	VirtualClusterContext string `json:"virtual_cluster_context"`
}
