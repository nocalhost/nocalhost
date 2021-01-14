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

package cluster

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"sync"
)

type ClusterStatus struct {
	ClusterId       uint64
	Ready           bool
	NotReadyMessage string
}

type ClusterSafeList struct {
	ClusterList []*model.ClusterList
	Lock        *sync.Mutex
}

// GetList Get the cluster list
// @Summary Get the cluster list
// @Description Get the cluster list
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.ClusterList "{"code":0,"message":"OK","data":model.ClusterList}"
// @Router /v1/cluster [get]
func GetList(c *gin.Context) {
	result, _ := service.Svc.ClusterSvc().GetList(c)
	// check if dep is ready
	//if err != nil || len(result) > 0 {
	//	userClusterList := &ClusterSafeList{
	//		Lock:        new(sync.Mutex),
	//		ClusterList: make([]*model.ClusterList, 0),
	//	}
	//	fmt.Printf("clusterlist %s", userClusterList.ClusterList)
	//	wait := sync.WaitGroup{}
	//	//clusterStatus := make(chan ClusterStatus, len(result))
	//	// check point
	//	// 1. has nocalhost-reserved NS
	//	// 2. has nocalhost-dep deployment
	//	// 3. nocalhost-dep deployment is available
	//	// 4. all well means cluster Ready
	//	// not_ready_message always return one
	//	for _, cluster := range result {
	//		// use go func run all
	//		wait.Add(1)
	//		cluster := cluster
	//		go func(cluster *model.ClusterList, userClusterList *ClusterSafeList) {
	//			obj := &model.ClusterList{
	//				ID:          cluster.ID,
	//				ClusterName: cluster.ClusterName,
	//				UsersCount:  cluster.UsersCount,
	//				KubeConfig:  cluster.KubeConfig,
	//				Info:        cluster.Info,
	//				UserId:      cluster.UserId,
	//				CreatedAt:   cluster.CreatedAt,
	//				IsReady:     true,
	//			}
	//			defer wait.Done()
	//			userClusterList.Lock.Lock()
	//			defer userClusterList.Lock.Unlock()
	//			clientGo, err := clientgo.NewGoClient([]byte(cluster.KubeConfig))
	//			if err != nil {
	//				log.Warnf("create go-client when get cluster list err %s", clientGo)
	//				obj.IsReady = false
	//				obj.NotReadyMessage = "New go client fail"
	//				userClusterList.ClusterList = append(userClusterList.ClusterList, obj)
	//				return
	//			}
	//			_, err = clientGo.IfNocalhostNameSpaceExist()
	//			if err != nil {
	//				obj.IsReady = false
	//				obj.NotReadyMessage = "Can not get namespace: " + global.NocalhostSystemNamespace
	//				userClusterList.ClusterList = append(userClusterList.ClusterList, obj)
	//				return
	//			}
	//			err = clientGo.GetDepDeploymentStatus()
	//			if err != nil {
	//				obj.IsReady = false
	//				obj.NotReadyMessage = err.Error()
	//				userClusterList.ClusterList = append(userClusterList.ClusterList, obj)
	//				return
	//			}
	//			userClusterList.ClusterList = append(userClusterList.ClusterList, obj)
	//		}(cluster, userClusterList)
	//	}
	//
	//	wait.Wait()
	//	api.SendResponse(c, errno.OK, userClusterList.ClusterList)
	//	return
	//}

	api.SendResponse(c, errno.OK, result)
}

// @Summary Cluster dev space list
// @Description Cluster entrance to obtain cluster development environment
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Cluster ID"
// @Success 200 {object} model.ClusterUserModel "kubeconfig"
// @Router /v1/cluster/{id}/dev_space [get]
func GetSpaceList(c *gin.Context) {
	//userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	where := model.ClusterUserModel{
		ClusterId: clusterId,
	}
	result, err := service.Svc.ClusterUser().GetList(c, where)
	if err != nil {
		api.SendResponse(c, nil, make([]interface{}, 0))
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get cluster details
// @Description Get cluster details
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Cluster ID"
// @Success 200 {object} model.ClusterModel "include kubeconfig"
// @Router /v1/cluster/{id}/detail [get]
func GetDetail(c *gin.Context) {
	// userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ClusterSvc().Get(c, clusterId)

	if err != nil {
		api.SendResponse(c, nil, make([]interface{}, 0))
		return
	}

	resp := ClusterDetailResponse{
		ID:           result.ID,
		Name:         result.Name,
		Info:         result.Info,
		UserId:       result.UserId,
		Server:       result.Server,
		KubeConfig:   "",
		StorageClass: result.StorageClass,
		CreatedAt:    result.CreatedAt,
	}

	// recreate
	//clusterDetail := model.ClusterDetailModel{
	//	ID:              result.ID,
	//	Name:            result.Name,
	//	Info:            result.Info,
	//	UserId:          result.UserId,
	//	Server:          result.Server,
	//	KubeConfig:      result.KubeConfig,
	//	CreatedAt:       result.CreatedAt,
	//	UpdatedAt:       result.UpdatedAt,
	//	DeletedAt:       result.DeletedAt,
	//	IsReady:         true,
	//	NotReadyMessage: "",
	//}
	//
	//// check cluster status
	//clientGo, err := clientgo.NewGoClient([]byte(result.KubeConfig))
	//if err != nil {
	//	clusterDetail.NotReadyMessage = "New go client fail"
	//	clusterDetail.IsReady = false
	//	api.SendResponse(c, nil, clusterDetail)
	//	return
	//}
	//_, err = clientGo.IfNocalhostNameSpaceExist()
	//if err != nil {
	//	clusterDetail.NotReadyMessage = "Can not get namespace: " + global.NocalhostSystemNamespace
	//	clusterDetail.IsReady = false
	//	api.SendResponse(c, nil, clusterDetail)
	//	return
	//}
	//err = clientGo.GetDepDeploymentStatus()
	//if err != nil {
	//	clusterDetail.NotReadyMessage = err.Error()
	//	clusterDetail.IsReady = false
	//	api.SendResponse(c, nil, clusterDetail)
	//	return
	//}

	api.SendResponse(c, errno.OK, resp)
}

// @Summary Details of a development environment in the cluster
// @Description Get cluster development environment details through cluster id and development environment id
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Cluster ID"
// @Param space_id path string true "DevSpace ID"
// @Success 200 {object} model.ClusterUserModel "include kubeconfig"
// @Router /v1/cluster/{id}/dev_space/{space_id}/detail [get]
func GetSpaceDetail(c *gin.Context) {
	//userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	devSpaceId := cast.ToUint64(c.Param("space_id"))
	where := model.ClusterUserModel{
		ID:        devSpaceId,
		ClusterId: clusterId,
	}
	result, err := service.Svc.ClusterUser().GetFirst(c, where)
	if err != nil {
		api.SendResponse(c, nil, nil)
		return
	}
	api.SendResponse(c, nil, result)
}

// @Summary Get cluster storageClass from cluster list
// @Description Get cluster storageClass from cluster list
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "Cluster ID"
// @Success 200 {object} cluster.StorageClassResponse "include kubeconfig"
// @Router /v1/cluster/{id}/storage_class [get]
func GetStorageClass(c *gin.Context) {
	// userId, _ := c.Get("userId")
	clusterKey := c.Param("id")
	var kubeConfig []byte
	if clusterKey == "kubeconfig" {
		var req StorageClassRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			api.SendResponse(c, errno.ErrBind, nil)
			return
		}
		if req.KubeConfig == "" {
			api.SendResponse(c, errno.ErrParam, nil)
			return
		}
		var err error
		if req.KubeConfig != "" {
			kubeConfig, err = base64.StdEncoding.DecodeString(req.KubeConfig)
			if err != nil {
				api.SendResponse(c, errno.ErrClusterKubeErr, nil)
				return
			}
		}
	} else {
		cluster, err := service.Svc.ClusterSvc().Get(c, cast.ToUint64(clusterKey))
		if err != nil {
			api.SendResponse(c, errno.ErrClusterNotFound, nil)
			return
		}
		kubeConfig = []byte(cluster.KubeConfig)
	}

	// new client go
	clientGo, err := clientgo.NewAdminGoClient(kubeConfig)
	if err != nil {
		api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		return
	}
	storageClassList, err := clientGo.GetStorageClassList()
	if err != nil {
		api.SendResponse(c, errno.ErrGetClusterStorageClass, nil)
		return
	}
	var typeName []string
	for _, st := range storageClassList.Items {
		typeName = append(typeName, st.Name)
	}
	response := StorageClassResponse{
		TypeName: typeName,
	}
	api.SendResponse(c, nil, response)
	return
}

// @Summary Get cluster storageClass from create cluster
// @Description Get cluster storageClass from create cluster
// @Tags Cluster
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param createCluster body cluster.StorageClassRequest true "The cluster info"
// @Success 200 {object} cluster.StorageClassResponse "include kubeconfig"
// @Router /v1/cluster/kubeconfig/storage_class [post]
func GetStorageClassByKubeConfig(c *gin.Context) {
	GetStorageClass(c)
}
