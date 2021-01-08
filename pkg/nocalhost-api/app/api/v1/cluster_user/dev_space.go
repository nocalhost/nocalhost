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
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
)

type DevSpace struct {
	DevSpaceParams ClusterUserCreateRequest
	c              *gin.Context
	KubeConfig     []byte
}

func NewDevSpace(devSpaceParams ClusterUserCreateRequest, c *gin.Context, kubeConfig []byte) *DevSpace {
	return &DevSpace{
		DevSpaceParams: devSpaceParams,
		c:              c,
		KubeConfig:     kubeConfig,
	}
}

func (d *DevSpace) Delete() error {
	goClient, err := clientgo.NewGoClient(d.KubeConfig)
	if err != nil {
		return errno.ErrClusterKubeErr
	}
	_, _ = goClient.DeleteNS(d.DevSpaceParams.NameSpace)

	// delete database cluster-user dev space
	dErr := service.Svc.ClusterUser().Delete(d.c, *d.DevSpaceParams.ID)
	if dErr != nil {
		return errno.ErrDeletedClsuterButDatabaseFail
	}
	return nil
}

func (d *DevSpace) Create() (*model.ClusterUserModel, error) {
	userId := cast.ToUint64(d.DevSpaceParams.UserId)
	applicationId := cast.ToUint64(d.DevSpaceParams.ApplicationId)
	// get user
	usersRecord, err := service.Svc.UserSvc().GetUserByID(d.c, userId)
	if err != nil {
		return nil, errno.ErrUserNotFound
	}

	// check application
	applicationRecord, err := service.Svc.ApplicationSvc().Get(d.c, applicationId)
	if err != nil {
		return nil, errno.ErrPermissionApplication
	}

	var decodeApplicationJson map[string]interface{}
	err = json.Unmarshal([]byte(applicationRecord.Context), &decodeApplicationJson)
	if err != nil {
		return nil, errno.ErrApplicationJsonContext
	}

	applicationName := ""
	if decodeApplicationJson["application_name"] != nil {
		applicationName = decodeApplicationJson["application_name"].(string)
	}

	spaceName := applicationName + "[" + usersRecord.Name + "]"
	if d.DevSpaceParams.SpaceName != "" {
		spaceName = d.DevSpaceParams.SpaceName
	}

	// check cluster
	clusterData, err := service.Svc.ClusterSvc().Get(d.c, *d.DevSpaceParams.ClusterId)
	if err != nil {
		return nil, errno.ErrPermissionCluster
	}
	// check if has auth
	cu := model.ClusterUserModel{
		ApplicationId: applicationId,
		UserId:        userId,
	}
	_, hasRecord := service.Svc.ClusterUser().GetFirst(d.c, cu)
	if hasRecord == nil {
		return nil, errno.ErrBindUserClsuterRepeat
	}

	// create namespace
	var KubeConfig = []byte(clusterData.KubeConfig)
	goClient, err := clientgo.NewGoClient(KubeConfig)
	if err != nil {
		return nil, errno.ErrClusterKubeErr
	}
	// create cluster devs
	devNamespace := goClient.GenerateNsName(userId)
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)
	secret, err := clusterDevsSetUp.CreateNS(devNamespace, "").CreateServiceAccount("", devNamespace).CreateRole(global.NocalhostDevRoleName, devNamespace).CreateRoleBinding(global.NocalhostDevRoleBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevServiceAccountName).CreateRoleBinding(global.NocalhostDevRoleDefaultBindingName, devNamespace, global.NocalhostDevRoleName, global.NocalhostDevDefaultServiceAccountName).GetServiceAccount(global.NocalhostDevServiceAccountName, devNamespace).GetServiceAccountSecret("", devNamespace)
	KubeConfigYaml, err, nerrno := setupcluster.NewDevKubeConfigReader(secret, clusterData.Server, devNamespace).GetCA().GetToken().AssembleDevKubeConfig().ToYamlString()
	if err != nil {
		return nil, nerrno
	}

	result, err := service.Svc.ClusterUser().Create(d.c, applicationId, *d.DevSpaceParams.ClusterId, userId, *d.DevSpaceParams.Memory, *d.DevSpaceParams.Cpu, KubeConfigYaml, devNamespace, spaceName)
	if err != nil {
		return nil, errno.ErrBindApplicationClsuter
	}
	return &result, nil
}
