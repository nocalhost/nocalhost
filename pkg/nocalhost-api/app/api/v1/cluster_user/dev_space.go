/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"math/rand"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"nocalhost/pkg/nocalhost-api/pkg/setupcluster"
	"time"
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
	goClient, err := clientgo.NewAdminGoClient(d.KubeConfig)

	// get client go and check if is admin Kubeconfig
	if err != nil {
		switch err.(type) {
		case *errno.Errno:
			return err
		default:
			return errno.ErrClusterKubeErr
		}
	}

	// delete tracing header from base space
	if d.DevSpaceParams.BaseDevSpaceId > 0 {
		if err := d.deleteTracingHeader(); err != nil {
			log.Error(err)
			return errno.ErrDeleteTracingHeaderFailed
		}
	}

	_, _ = goClient.DeleteNS(d.DevSpaceParams.NameSpace)

	// delete database cluster-user dev space
	dErr := service.Svc.ClusterUserSvc.Delete(d.c, *d.DevSpaceParams.ID)
	if dErr != nil {
		return errno.ErrDeletedClusterButDatabaseFail
	}
	return nil
}

func (d *DevSpace) Create() (*model.ClusterUserModel, error) {
	userId := cast.ToUint64(d.DevSpaceParams.UserId)
	clusterId := cast.ToUint64(d.DevSpaceParams.ClusterId)

	// get user
	usersRecord, err := service.Svc.UserSvc.GetUserByID(d.c, userId)
	if err != nil {
		return nil, errno.ErrUserNotFound
	}

	// check cluster
	clusterRecord, err := service.Svc.ClusterSvc.Get(context.TODO(), clusterId)
	if err != nil {
		return nil, errno.ErrClusterNotFound
	}

	if d.DevSpaceParams.SpaceName == "" {
		if genName, err := getUnDuplicateName(
			0, fmt.Sprintf("%s[%s]", clusterRecord.Name, usersRecord.Name),
		); err != nil {
			return nil, err
		} else {
			d.DevSpaceParams.SpaceName = genName
		}
	} else {
		// check if space name exist
		if _, err := service.Svc.ClusterUserSvc.GetFirst(
			d.c, model.ClusterUserModel{
				SpaceName: d.DevSpaceParams.SpaceName,
			},
		); err == nil {
			return nil, errno.ErrSpaceNameAlreadyExists
		}
	}

	// check base dev space
	baseClusterUser := &model.ClusterUserModel{}
	if d.DevSpaceParams.BaseDevSpaceId > 0 {
		var err error
		baseClusterUser, err = service.Svc.ClusterUserSvc.GetFirst(
			d.c, model.ClusterUserModel{
				ID: d.DevSpaceParams.BaseDevSpaceId,
			},
		)
		if err != nil || baseClusterUser == nil {
			log.Error(err)
			return nil, errno.ErrMeshClusterUserNotFound
		}
		if baseClusterUser.BaseDevSpaceId > 0 {
			return nil, errno.ErrUseAsBaseSpace
		}
		if baseClusterUser.Namespace == "*" || baseClusterUser.Namespace == "" {
			log.Error(errors.New("base dev namespace has not found"))
			return nil, errno.ErrMeshClusterUserNamespaceNotFound
		}
	}

	clusterUserModel := &model.ClusterUserModel{}
	if d.DevSpaceParams.ClusterAdmin == nil || *d.DevSpaceParams.ClusterAdmin == 0 {
		clusterUserModel, err = d.createDevSpace(clusterRecord, usersRecord)
		if err != nil {
			return nil, err
		}
	} else {
		clusterUserModel, err = d.createClusterDevSpace(clusterRecord, usersRecord)
		if err != nil {
			return nil, err
		}
	}

	// init mesh dev space
	if d.DevSpaceParams.BaseDevSpaceId > 0 {
		if clusterUserModel, err = d.initMeshDevSpace(&clusterRecord, clusterUserModel, baseClusterUser); err != nil {
			log.Error(err)
			// rollback
			log.Debugf("rollback dev space %s", clusterUserModel.SpaceName)
			d.KubeConfig = []byte(clusterRecord.GetKubeConfig())
			d.DevSpaceParams.NameSpace = clusterUserModel.Namespace
			d.DevSpaceParams.SpaceName = clusterUserModel.SpaceName
			d.DevSpaceParams.ID = &clusterUserModel.ID
			_ = d.Delete()
			return nil, errno.ErrInitMeshSpaceFailed
		}
	}

	return clusterUserModel, nil
}

func getUnDuplicateName(times int, name string) (string, error) {
	spaceName := name
	if times > 0 {
		spaceName += random()
	}

	// check if space name exist
	if _, err := service.Svc.ClusterUserSvc.GetFirst(
		context.TODO(), model.ClusterUserModel{
			SpaceName: spaceName,
		},
	); err == nil {
		if times < 3 {
			return getUnDuplicateName(times+1, name)
		} else {
			return "", errno.ErrSpaceNameAlreadyExists
		}
	}

	return spaceName, nil
}

func random() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, 4)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (d *DevSpace) createClusterDevSpace(
	clusterRecord model.ClusterModel, usersRecord *model.UserBaseModel,
) (*model.ClusterUserModel, error) {
	trueFlag := uint64(1)
	list, err := service.Svc.ClusterUserSvc.GetList(
		context.TODO(), model.ClusterUserModel{
			ClusterId:    clusterRecord.ID,
			UserId:       usersRecord.ID,
			ClusterAdmin: &trueFlag,
		},
	)
	if len(list) > 0 {
		return nil, errno.ErrAlreadyExist
	}

	result, err := service.Svc.ClusterUserSvc.CreateClusterAdminSpace(
		context.TODO(), clusterRecord.ID, usersRecord.ID, d.DevSpaceParams.SpaceName,
	)
	if err != nil {
		return nil, errno.ErrBindApplicationClsuter
	}

	if err := service.Svc.AuthorizeClusterToUser(clusterRecord.ID, usersRecord.ID); err != nil {
		return nil, err
	}

	return &result, nil
}

// createDevSpace
// create a devSpace, if already exist, return the
// devSpace created before
func (d *DevSpace) createDevSpace(
	clusterRecord model.ClusterModel, usersRecord *model.UserBaseModel,
) (*model.ClusterUserModel, error) {

	applicationId := cast.ToUint64(d.DevSpaceParams.ApplicationId)

	// create namespace
	var KubeConfig = []byte(clusterRecord.KubeConfig)
	goClient, err := clientgo.NewAdminGoClient(KubeConfig)

	// get client go and check if is admin Kubeconfig
	if err != nil {
		switch err.(type) {
		case *errno.Errno:
			return nil, err
		default:
			return nil, errno.ErrClusterKubeErr
		}
	}

	// (1) if namespace is specified, no need to create this namespace
	// but we should initial the namespace
	needCreateNamespace := true
	devNamespace := goClient.GenerateNsName(usersRecord.ID)

	if d.DevSpaceParams.NameSpace != "" {
		devNamespace = d.DevSpaceParams.NameSpace

		_, err := goClient.GetNamespace(devNamespace)
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, err
		} else {
			needCreateNamespace = false
		}
	}

	// create cluster devs
	clusterDevsSetUp := setupcluster.NewClusterDevsSetUp(goClient)

	// (2) process for mesh space
	// set labels for istio proxy sidecar injection
	labels := make(map[string]string)
	if d.DevSpaceParams.BaseDevSpaceId > 0 {
		if d.DevSpaceParams.MeshDevInfo == nil {
			return nil, errno.ErrMeshInfoRequired
		}
		if d.DevSpaceParams.MeshDevInfo.MeshDevNamespace != "" && !d.DevSpaceParams.MeshDevInfo.ReCreate {
			devNamespace = d.DevSpaceParams.MeshDevInfo.MeshDevNamespace
		}
		labels["istio-injection"] = "enabled"
		labels["nocalhost.dev/devspace"] = "share"

		if d.DevSpaceParams.MeshDevInfo.Header.TraceType == "jaeger" {
			d.DevSpaceParams.MeshDevInfo.Header.TraceKey = "uberctx-trace"
			d.DevSpaceParams.MeshDevInfo.Header.TraceValue = devNamespace
		}
		if d.DevSpaceParams.MeshDevInfo.Header.TraceType == "zipkin" {
			d.DevSpaceParams.MeshDevInfo.Header.TraceKey = "baggage-trace"
			d.DevSpaceParams.MeshDevInfo.Header.TraceValue = devNamespace
		}
	}

	if d.DevSpaceParams.IsBaseSpace {
		labels["istio-injection"] = "enabled"
		labels["nocalhost.dev/devspace"] = "base"
	}

	// (3) create the devspace
	if needCreateNamespace {
		// create namespace
		_, err = goClient.CreateNS(devNamespace, labels)
		if err != nil {
			return nil, errno.ErrNameSpaceCreate
		}
	}

	// (4) initial the devspace
	// create namespace ResourceQuota and container limitRange
	res := d.DevSpaceParams.SpaceResourceLimit
	if res == nil {
		res = &SpaceResourceLimit{}
	}

	clusterDevsSetUp.CreateResourceQuota(
		"rq-"+devNamespace, devNamespace, res.SpaceReqMem,
		res.SpaceReqCpu, res.SpaceLimitsMem, res.SpaceLimitsCpu, res.SpaceStorageCapacity, res.SpaceEphemeralStorage,
		res.SpacePvcCount, res.SpaceLbCount,
	).CreateLimitRange(
		"lr-"+devNamespace, devNamespace,
		res.ContainerReqMem, res.ContainerLimitsMem, res.ContainerReqCpu, res.ContainerLimitsCpu,
		res.ContainerEphemeralStorage,
	)

	var result model.ClusterUserModel

	if any, _ := service.Svc.ClusterUserSvc.GetFirst(
		context.TODO(), model.ClusterUserModel{
			Namespace: devNamespace,
		},
	); any != nil {
		result = *any
	} else {
		resString, err := json.Marshal(res)
		result, err = service.Svc.ClusterUserSvc.Create(
			d.c, *d.DevSpaceParams.ClusterId, usersRecord.ID, *d.DevSpaceParams.Memory, *d.DevSpaceParams.Cpu,
			"", devNamespace, d.DevSpaceParams.SpaceName, string(resString), d.DevSpaceParams.IsBaseSpace,
			d.DevSpaceParams.Protected,
		)
		if err != nil {
			return nil, errno.ErrBindApplicationClsuter
		}
	}

	// auth application to user
	_ = service.Svc.ApplicationUserSvc.BatchInsert(d.c, applicationId, []uint64{usersRecord.ID})

	// authorize namespace to user
	if err := service.Svc.AuthorizeNsToUser(clusterRecord.ID, usersRecord.ID, result.Namespace); err != nil {
		return nil, err
	}

	if err := service.Svc.AuthorizeNsToDefaultSa(clusterRecord.ID, usersRecord.ID, result.Namespace); err != nil {
		return nil, err
	}

	return &result, nil
}

func (d *DevSpace) initMeshDevSpace(
	clusterRecord *model.ClusterModel, clusterUser, baseClusterUser *model.ClusterUserModel) (
	*model.ClusterUserModel, error) {
	// init mesh dev space
	meshDevInfo := d.DevSpaceParams.MeshDevInfo
	meshDevInfo.MeshDevNamespace = clusterUser.Namespace

	meshDevInfo.BaseNamespace = baseClusterUser.Namespace

	meshManager, err := setupcluster.GetSharedMeshManagerFactory().Manager(clusterRecord.KubeConfig)
	if err != nil {
		return clusterUser, err
	}

	if err := meshManager.InitMeshDevSpace(meshDevInfo); err != nil {
		_ = meshManager.Rollback(meshDevInfo)
		return clusterUser, err
	}

	clusterUser.TraceHeader = d.DevSpaceParams.MeshDevInfo.Header
	clusterUser.BaseDevSpaceId = d.DevSpaceParams.BaseDevSpaceId
	return service.Svc.ClusterUserSvc.Update(d.c, clusterUser)
}

func (d *DevSpace) deleteTracingHeader() error {
	if d.DevSpaceParams.BaseDevSpaceId == 0 {
		return nil
	}

	// check base dev space
	baseDevspace, err := service.Svc.ClusterUserSvc.GetFirst(
		d.c, model.ClusterUserModel{
			ID: d.DevSpaceParams.BaseDevSpaceId,
		},
	)
	if err != nil || baseDevspace == nil {
		log.Debug("can not find base namespace, does not delete tracing header")
		return nil
	}
	if baseDevspace.Namespace == "*" || baseDevspace.Namespace == "" {
		log.Debug("can not find base namespace, does not delete tracing header")
		return nil
	}

	meshDevInfo := d.DevSpaceParams.MeshDevInfo
	meshDevInfo.MeshDevNamespace = d.DevSpaceParams.NameSpace
	meshDevInfo.BaseNamespace = baseDevspace.Namespace

	meshManager, err := setupcluster.GetSharedMeshManagerFactory().Manager(string(d.KubeConfig))
	if err != nil {
		return err
	}

	if err := meshManager.DeleteTracingHeader(meshDevInfo); err != nil {
		return err
	}
	return nil
}
