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
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"nocalhost/internal/nocalhost-api/global"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"regexp"
	"strconv"
	"strings"
)

// Create Create a development environment for application
// @Summary Create a development environment for application
// @Description Create a development environment for application
// @Tags Application
// @Accept  json
// @Produce  json
// @param Authorization header string true "Authorization"
// @Param CreateAppRequest body cluster_user.ClusterUserCreateRequest true "cluster user info"
// @Param id path uint64 true "Application ID"
// @Success 200 {object} model.ClusterModel
// @Router /v1/application/{id}/create_space [post]
func Create(c *gin.Context) {
	var req ClusterUserCreateRequest
	defaultNum := uint64(0)
	req.Memory = &defaultNum
	req.Cpu = &defaultNum
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warnf("bind ApplicationCluster params err: %v", err)
		api.SendResponse(c, errno.ErrBind, nil)
		return
	}
	userId := cast.ToUint64(req.UserId)
	webUserId, _ := c.Get("userId")
	applicationId := cast.ToUint64(c.Param("id"))
	// get user
	usersRecord, err := service.Svc.UserSvc().GetUserByID(c, userId)
	if err != nil {
		api.SendResponse(c, errno.ErrUserNotFound, nil)
		return
	}

	// check application
	applicationRecord, err := service.Svc.ApplicationSvc().Get(c, applicationId, webUserId.(uint64))
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionApplication, nil)
		return
	}

	var decodeApplicationJson map[string]interface{}
	err = json.Unmarshal([]byte(applicationRecord.Context), &decodeApplicationJson)
	if err != nil {
		api.SendResponse(c, errno.ErrApplicationJsonContext, nil)
		return
	}

	applicationName := ""
	if decodeApplicationJson["application_name"] != nil {
		applicationName = decodeApplicationJson["application_name"].(string)
	}

	spaceName := applicationName + "[" + usersRecord.Name + "]"
	if req.SpaceName != "" {
		spaceName = req.SpaceName
	}

	// check cluster
	clusterData, err := service.Svc.ClusterSvc().Get(c, *req.ClusterId, webUserId.(uint64))
	if err != nil {
		api.SendResponse(c, errno.ErrPermissionCluster, nil)
		return
	}
	// check if has auth
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId,
	}
	record, hasRecord := service.Svc.ClusterUser().GetFirst(c, cu)
	if hasRecord == nil {
		log.Infof("cluster users %v", record)
		api.SendResponse(c, errno.ErrBindUserClsuterRepeat, nil)
		return
	}

	// Validate DevSpace Resource limit parameter format.
	flag, message := ValidSpaceResourceLimit(*req.SpaceResourceLimit)
	if !flag {
		log.Errorf("Create devspace fail. Incorrect Resource limit parameter  [ %v ] format.", message)
		api.SendResponse(c, errno.ErrFormatResourceLimitParam, message)
		return
	}
	// create namespace
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewGoClient(KubeConfig)
	if err != nil {
		log.Errorf("client go got err %v", err)
		api.SendResponse(c, errno.ErrClusterKubeErr, nil)
		return
	}
	// create cluster devs
	devNamespace := goClient.GenerateNsName(userId)
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)
	secret, err := clusterDevsSetUp.CreateNS(devNamespace, "").CreateServiceAccount("", devNamespace).CreateRole(global.NocalhostDevRoleName, devNamespace).CreateRoleBinding(global.NocalhostDevRoleBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevServiceAccountName).CreateRoleBinding(global.NocalhostDevRoleDefaultBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevDefaultServiceAccountName).GetServiceAccount(global.NocalhostDevServiceAccountName, devNamespace).GetServiceAccountSecret("", devNamespace)
	KubeConfigYaml, err, nerrno := setupcluster.NewDevKubeConfigReader(secret, clusterData.Server, devNamespace).GetCA().GetToken().AssembleDevKubeConfig().ToYamlString()
	if err != nil {
		api.SendResponse(c, nerrno, nil)
		return
	}

	// create namespace ResouceQuota and container limitRange
	res := req.SpaceResourceLimit
	clusterDevsSetUp.CreateResouceQuota("rq-"+devNamespace, devNamespace, res.SpaceReqMem,
		res.SpaceReqCpu, res.SpaceLimitsMem, res.SpaceLimitsCpu, res.SpaceStorageCapacity,
		res.SpacePvcCount, res.SpaceLbCount).CreateLimitRange("lr-"+devNamespace, devNamespace,
		res.ContainerReqMem, res.ContainerLimitsMem, res.ContainerReqCpu, res.ContainerLimitsCpu)

	resString, err := json.Marshal(req.SpaceResourceLimit)
	result, err := service.Svc.ClusterUser().Create(c, applicationId, *req.ClusterId, userId, *req.Memory, *req.Cpu, KubeConfigYaml, devNamespace, spaceName, string(resString))
	if err != nil {
		log.Warnf("create ApplicationCluster err: %v", err)
		api.SendResponse(c, errno.ErrBindApplicationClsuter, nil)
		return
	}

	api.SendResponse(c, nil, result)
}

func ValidSpaceResourceLimit(resLimit SpaceResourceLimit) (bool, string) {
	reg, _ := regexp.Compile("^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$")
	numReg, _ := regexp.Compile("^([+-]?[0-9]+)$")

	var message []string
	if resLimit.SpaceReqMem != "" && !reg.MatchString(resLimit.SpaceReqMem) {
		message = append(message, "space_req_mem")
	}
	if resLimit.SpaceLimitsMem != "" && !reg.MatchString(resLimit.SpaceLimitsMem) {
		message = append(message, "space_limits_mem")
	}
	if resLimit.SpaceReqCpu != "" && !reg.MatchString(resLimit.SpaceReqCpu) {
		message = append(message, "space_req_cpu")
	}
	if resLimit.SpaceLimitsCpu != "" && !reg.MatchString(resLimit.SpaceLimitsCpu) {
		message = append(message, "space_limits_cpu")
	}
	if resLimit.SpaceLbCount > 0 && !numReg.MatchString(strconv.Itoa(resLimit.SpaceLbCount)) {
		message = append(message, "space_lb_count")
	}
	if resLimit.SpacePvcCount > 0 && !numReg.MatchString(strconv.Itoa(resLimit.SpacePvcCount)) {
		message = append(message, "space_pvc_count")
	}
	if resLimit.SpaceStorageCapacity != "" && !reg.MatchString(resLimit.SpaceStorageCapacity) {
		message = append(message, "space_storage_capacity")
	}
	if resLimit.ContainerReqCpu != "" && !reg.MatchString(resLimit.ContainerReqCpu) {
		message = append(message, "container_req_cpu")
	}
	if resLimit.ContainerReqCpu != "" && !reg.MatchString(resLimit.ContainerReqCpu) {
		message = append(message, "container_req_cpu")
	}
	if resLimit.ContainerLimitsMem != "" && !reg.MatchString(resLimit.ContainerLimitsMem) {
		message = append(message, "container_limits_mem")
	}
	if resLimit.ContainerLimitsCpu != "" && !reg.MatchString(resLimit.ContainerLimitsCpu) {
		message = append(message, "container_limits_cpu")
	}
	if len(message) > 0 {
		return false, strings.Join(message, ",")
	}
	return true, strings.Join(message, ",")
}
