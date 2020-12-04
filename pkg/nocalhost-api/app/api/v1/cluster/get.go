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
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"sync"
)

type ClusterStatus struct {
	ClusterId       uint64
	Ready           bool
	NotReadyMessage string
}

// GetList 获取集群列表
// @Summary 获取集群列表
// @Description 获取集群列表
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Success 200 {object} model.ClusterList "{"code":0,"message":"OK","data":model.ClusterList}"
// @Router /v1/cluster [get]
func GetList(c *gin.Context) {
	result, err := service.Svc.ClusterSvc().GetList(c)
	// check if dep is ready
	if err != nil || len(result) > 0 {
		wait := sync.WaitGroup{}
		wait.Add(len(result))
		clusterStatus := make(chan ClusterStatus, len(result))
		// check point
		// 1. has nocalhost-reserved NS
		// 2. has nocalhost-dep deployment
		// 3. nocalhost-dep deployment is available
		// 4. all well means cluster Ready
		// not_ready_message always return one
		for _, cluster := range result {
			// use go func run all
			go func(cluster *model.ClusterList) {
				defer wait.Done()
				clientGo, err := clientgo.NewGoClient([]byte(cluster.KubeConfig))
				if err != nil {
					log.Warnf("create go-client when get cluster list err %s", clientGo)
					clusterStatus <- ClusterStatus{ClusterId: cluster.ID, Ready: false, NotReadyMessage: "New go client fail"}
					return
				}
				_, err = clientGo.IfNocalhostNameSpaceExist()
				if err != nil {
					clusterStatus <- ClusterStatus{ClusterId: cluster.ID, Ready: false, NotReadyMessage: "Can not get namespace: " + global.NocalhostSystemNamespace}
					return
				}
				err = clientGo.GetDepDeploymentStatus()
				if err != nil {
					clusterStatus <- ClusterStatus{ClusterId: cluster.ID, Ready: false, NotReadyMessage: err.Error()}
					return
				}
				clusterStatus <- ClusterStatus{ClusterId: cluster.ID, Ready: true, NotReadyMessage: ""}
			}(cluster)
		}

		fmt.Printf("got data %s", clusterStatus)
		// routine data
		fmt.Printf("len %d", cap(clusterStatus))

		go func() {
			for routineStatus := range clusterStatus {
				fmt.Printf("get channal data %s", routineStatus)
				for key, listRecord := range result {
					if routineStatus.ClusterId == listRecord.ID {
						result[key].IsReady = routineStatus.Ready
						result[key].NotReadyMessage = routineStatus.NotReadyMessage
						break
					}
				}
			}
		}()

		wait.Wait()
		close(clusterStatus)
	}

	api.SendResponse(c, nil, result)
}

// @Summary 集群开发环境列表
// @Description 集群入口获取集群开发环境
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "集群 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig"
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

// @Summary 获取集群详情
// @Description 获取集群详情
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "集群 ID"
// @Success 200 {object} model.ClusterModel "应用开发环境参数，含 kubeconfig"
// @Router /v1/cluster/{id}/detail [get]
func GetDetail(c *gin.Context) {
	userId, _ := c.Get("userId")
	clusterId := cast.ToUint64(c.Param("id"))
	result, err := service.Svc.ClusterSvc().Get(c, clusterId, userId.(uint64))

	if err != nil {
		api.SendResponse(c, nil, make([]interface{}, 0))
		return
	}

	// recreate
	clusterDetail := model.ClusterDetailModel{
		ID:              result.ID,
		Name:            result.Name,
		Info:            result.Info,
		UserId:          result.UserId,
		Server:          result.Server,
		KubeConfig:      result.KubeConfig,
		CreatedAt:       result.CreatedAt,
		UpdatedAt:       result.UpdatedAt,
		DeletedAt:       result.DeletedAt,
		IsReady:         true,
		NotReadyMessage: "",
	}

	// check cluster status
	clientGo, err := clientgo.NewGoClient([]byte(result.KubeConfig))
	if err != nil {
		clusterDetail.NotReadyMessage = "New go client fail"
		clusterDetail.IsReady = false
		api.SendResponse(c, nil, clusterDetail)
		return
	}
	_, err = clientGo.IfNocalhostNameSpaceExist()
	if err != nil {
		clusterDetail.NotReadyMessage = "Can not get namespace: " + global.NocalhostSystemNamespace
		clusterDetail.IsReady = false
		api.SendResponse(c, nil, clusterDetail)
		return
	}
	err = clientGo.GetDepDeploymentStatus()
	if err != nil {
		clusterDetail.NotReadyMessage = err.Error()
		clusterDetail.IsReady = false
		api.SendResponse(c, nil, clusterDetail)
		return
	}

	api.SendResponse(c, nil, clusterDetail)
}

// @Summary 集群某个开发环境的详情
// @Description 通过集群 id 和开发环境 id 获取集群开发环境详情
// @Tags 集群
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path string true "集群 ID"
// @Param space_id path string true "开发空间 ID"
// @Success 200 {object} model.ClusterUserModel "应用开发环境参数，含 kubeconfig"
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
