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

package cluster_user

import (
	"github.com/gin-gonic/gin"
	reqClient "github.com/imroc/req"
	"github.com/spf13/cast"
	"net/http"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strconv"
	"time"
)

// Delete Completely delete the development environment
// @Summary Completely delete the development environment
// @Description Completely delete the development environment, including deleting the K8S namespace
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} api.Response "{"code":0,"message":"OK","data":null}"
// @Router /v1/dev_space/{id} [delete]
func Delete(c *gin.Context) {
	userId, _ := c.Get("userId")
	devSpaceId := cast.ToUint64(c.Param("id"))
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClsuterUserNotFound, nil)
		return
	}
	clusterData, err := service.Svc.ClusterSvc().Get(c, clusterUser.ClusterId, userId.(uint64))
	if err != nil {
		api.SendResponse(c, errno.ErrClusterNotFound, nil)
		return
	}

	// client go and delete specify namespace
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewGoClient(KubeConfig)
	if err != nil {
		log.Errorf("client go got err %v", err)
		api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		return
	}
	isDelete, _ := goClient.DeleteNS(clusterUser.Namespace)
	if !isDelete {
		// ignore deleteNS and should delete dev space record
		log.Infof("delete cluster user, and try delete cluster dev space %s fail", clusterUser.Namespace)
	}

	// delete database cluster-user dev space
	dErr := service.Svc.ClusterUser().Delete(c, clusterUser.ID)
	if dErr != nil {
		api.SendResponse(c, errno.ErrDeletedClsuterButDatabaseFail, nil)
		return
	}
	api.SendResponse(c, errno.OK, nil)
	return
}

// ReCreate ReCreate devSpace
// @Summary ReCreate devSpace
// @Description delete devSpace and create a new one
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/dev_space/{id}/recreate [post]
func ReCreate(c *gin.Context) {
	// get devSpace
	devSpaceId := cast.ToUint64(c.Param("id"))
	clusterUser, err := service.Svc.ClusterUser().GetFirst(c, model.ClusterUserModel{ID: devSpaceId})
	if err != nil {
		api.SendResponse(c, errno.ErrClsuterUserNotFound, nil)
		return
	}
	auth := c.Request.Header["Authorization"][0]
	header := reqClient.Header{
		"Accept":        "application/json",
		"Authorization": auth,
	}
	reqClient.SetTimeout(60 * time.Second)
	protocol := "http://"
	host := c.Request.Host
	_, err = reqClient.Get(protocol + host + "/health")
	// if timeout, means protocol fail
	if err != nil {
		protocol = "https://"
	}
	// delete devSpace space first, it will delete database record whatever success delete namespace or not
	_, _ = reqClient.Delete(protocol+host+"/v1/dev_space/"+strconv.Itoa(int(devSpaceId)), header)
	// create new devSpace
	param := reqClient.Param{
		"cluster_id": clusterUser.ClusterId,
		"cpu":        clusterUser.Cpu,
		"memory":     clusterUser.Memory,
		"space_name": clusterUser.SpaceName,
		"user_id":    clusterUser.UserId,
	}
	result, err := reqClient.Post(protocol+host+"/v1/application/"+strconv.Itoa(int(clusterUser.ApplicationId))+"/create_space", header, reqClient.BodyJSON(&param))
	var returnResult interface{}
	_ = result.ToJSON(&returnResult)
	c.JSON(http.StatusOK, &returnResult)
	return
}

// ReCreate Plugin ReCreate devSpace
// @Summary Plugin ReCreate devSpace
// @Description Plugin delete devSpace and create a new one
// @Tags Plug-in
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param id path uint64 true "DevSpace ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/plugin/{id}/recreate [post]
func PluginReCreate(c *gin.Context) {
	ReCreate(c)
}
