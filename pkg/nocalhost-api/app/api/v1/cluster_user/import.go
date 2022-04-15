/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_user

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/internal/nocalhost-api/service/cooperator/cluster_scope"
	"nocalhost/internal/nocalhost-api/service/cooperator/ns_scope"
	"nocalhost/pkg/nocalhost-api/app/api"
	"nocalhost/pkg/nocalhost-api/app/router/ginbase"
	"nocalhost/pkg/nocalhost-api/app/router/middleware"
	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
	"nocalhost/pkg/nocalhost-api/pkg/log"
	"strings"
	"sync"
)

type NamespaceInfo struct {
	Name         string
	Cluster      string
	IstioEnabled int
}

func GetNsInfo(c *gin.Context) {

	allClusters, _ := service.Svc.ClusterSvc.GetList(c)
	tempResult := make([]*model.ClusterList, 0, 0)
	userId := c.GetUint64("userId")
	// normal user can only see clusters they created, or devSpace's cluster
	if isAdmin, _ := middleware.IsAdmin(c); !isAdmin {
		// cluster --> userid, find cluster which user's devSpace based on
		clusterToUser := make(map[uint64]uint64)
		// get clusters which associated with current user, like cluster which current user's devSpace based on
		lists, _ := DoList(&model.ClusterUserModel{}, userId, false, false)
		for _, i := range lists {
			clusterToUser[i.ClusterId] = i.ClusterId
		}
		for _, list := range allClusters {
			// cluster they created, can modify
			if list.UserId == userId {
				list.Modifiable = true
				tempResult = append(tempResult, list)
				// cluster devSpace based on, can't modify
			} else if _, ok := clusterToUser[list.GetClusterId()]; ok {
				list.Modifiable = false
				tempResult = append(tempResult, list)
			}
		}
		allClusters = tempResult[0:]
	} else {
		// administer have all privilege
		for _, list := range allClusters {
			list.Modifiable = true
		}
	}

	cum := model.ClusterUserModel{}
	devSpaces, err := DoList(&cum, userId, true, false)
	if err != nil && !errors.Is(err, errno.ErrClusterNotFound) {
		api.SendResponse(c, err, "")
		return
	}

	nsList := make([]*NamespaceInfo, 0)
	for _, list := range allClusters {
		goClient, err := clientgo.NewAdminGoClient([]byte(list.KubeConfig))
		if err != nil {
			log.Error(err.Error())
			continue
		}
		istioEnabled, _ := goClient.CheckIstio()
		nss, err := goClient.GetClientSet().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Error(err.Error())
		}
		if len(nss.Items) == 0 {
			log.Infof("No namespace found in %s", list.ClusterName)
			continue
		}
		for _, item := range nss.Items {
			var exist bool
			for _, space := range devSpaces {
				if space.ClusterName == list.ClusterName && space.Namespace == item.Name {
					exist = true
					break
				}
			}
			if exist {
				continue
			}
			nsi := NamespaceInfo{
				Name:         item.Name,
				Cluster:      list.ClusterName,
				IstioEnabled: 0,
			}
			if istioEnabled {
				nsi.IstioEnabled = 1
			}
			nsList = append(nsList, &nsi)
		}
	}
	api.SendResponse(c, nil, nsList)
}

type NsImportItem struct {
	ClusterName  string   `json:"cluster_name"`
	Namespace    string   `json:"namespace"`
	Owner        *uint64  `json:"owner"`
	IsBaseSpace  int      `json:"is_basespace"`
	Collaborator []uint64 `json:"collaborator"`
}

func importNsToDevSpace(ctx *gin.Context, devSpace []*BatchImportItem, uuid string) {
	its, _ := nsImportStatusMap.Load(uuid)
	if its == nil {
		log.Errorf("no ns import status %s found in nsImportStatusMap", uuid)
		return
	}

	userId, _ := ginbase.LoginUser(ctx)

	nis, _ := its.(*NsImportStatus)

	for i, item := range devSpace {
		nisi := NsImportStatusItem{}
		nisi.BatchImportItem = *item

		var uid uint64
		var err error
		if item.Owner == "" {
			uid = userId
		} else {
			var u *model.UserBaseModel
			u, err = service.Svc.UserSvc.GetUserByEmail(context.TODO(), item.Owner)
			if err == nil {
				uid = u.ID
			}
		}

		if err == nil {
			cs := make([]uint64, 0)
			for _, s := range item.Collaborator {
				var uu *model.UserBaseModel
				uu, err = service.Svc.UserSvc.GetUserByEmail(context.TODO(), s)
				if err != nil {
					err = errors.New(fmt.Sprintf("Get user %s failed: %s", s, err.Error()))
					break
				}
				cs = append(cs, uu.ID)
			}

			if err == nil {
				req := NsImportItem{
					ClusterName:  item.ClusterName,
					Namespace:    item.Namespace,
					Owner:        &uid,
					IsBaseSpace:  0,
					Collaborator: cs,
				}
				err = importNs(ctx, &req)
			}
		}

		if err != nil {
			nisi.ErrInfo = err.Error()
			//if strings.Contains(err.Error(), "record not found") {
			//	nisi.ErrInfo = fmt.Sprintf("importing %s(owner:%s) failed: %s", item.Namespace, item.Owner, err.Error())
			//}
		} else {
			nisi.Success = true
		}
		nis.Process = (float32(i) + 1.0) / float32(len(devSpace))
		nis.Items = append(nis.Items, &nisi)
	}
}

func importNs(c1 *gin.Context, req *NsImportItem) error {
	if req.ClusterName == "" {
		return errors.New("cluster_name can not be nil")
	}

	var cucr ClusterUserCreateRequest

	var clusterId uint64
	clusters, _ := service.Svc.ClusterSvc.GetList(context.TODO())
	var cluster *model.ClusterList
	for _, list := range clusters {
		if list.ClusterName == req.ClusterName {
			clusterId = list.ID
			cluster = list
		}
	}

	if clusterId == 0 {
		return errors.New(fmt.Sprintf("cluster %s not found", req.ClusterName))
	}

	if req.Namespace == "" {
		return errors.New("namespace can not be nil")
	}

	if req.Owner == nil {
		return errors.New("owner can not be nil")
	}

	goClient, err := clientgo.NewAdminGoClient([]byte(cluster.KubeConfig))
	if err != nil {
		return err
	}
	nss, err := goClient.GetClientSet().CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	var nsExist bool
	for _, item := range nss.Items {
		if item.Name == req.Namespace {
			nsExist = true
			break
		}
	}
	if !nsExist {
		return errors.New(fmt.Sprintf("Namespace %s not found in %s", req.Namespace, cluster.ClusterName))
	}

	applicationId := uint64(0)
	cucr.ApplicationId = &applicationId
	cucr.NameSpace = req.Namespace
	cucr.SpaceName = req.Namespace
	cucr.ClusterId = &clusterId
	cucr.UserId = req.Owner
	clusterAdmin := uint64(0)
	defaultNum := uint64(0)
	cucr.Memory = &defaultNum
	cucr.Cpu = &defaultNum
	cucr.ClusterAdmin = &clusterAdmin
	if req.IsBaseSpace > 0 {
		cucr.IsBaseSpace = true
	}
	devSpace := NewDevSpace(cucr, c1, []byte{})
	spaceInfo, err := devSpace.Create()
	if err != nil {
		return err
	}
	var errStr string
	cu, err := HasModifyPermissionToSomeDevSpace(*cucr.UserId, spaceInfo.ID)
	if err != nil {
		return err
	}

	// resource limit
	var srl *SpaceResourceLimit
	rqs, err := goClient.GetClientSet().CoreV1().ResourceQuotas(req.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range rqs.Items {
		if item.Spec.ScopeSelector != nil || len(item.Spec.Scopes) != 0 {
			continue
		}
		resourceList := item.Spec.Hard
		if len(resourceList) > 0 {
			if srl == nil {
				srl = &SpaceResourceLimit{}
			}
			if rs, ok := resourceList[corev1.ResourceRequestsMemory]; ok {
				srl.SpaceReqMem = fmt.Sprintf("%dMi", rs.Value()>>20)
			}
			if rs, ok := resourceList[corev1.ResourceRequestsCPU]; ok {
				srl.SpaceReqCpu = rs.String()
			}
			if rs, ok := resourceList[corev1.ResourceLimitsMemory]; ok {
				srl.SpaceLimitsMem = fmt.Sprintf("%dMi", rs.Value()>>20)
			}
			if rs, ok := resourceList[corev1.ResourceLimitsCPU]; ok {
				srl.SpaceLimitsCpu = rs.String()
			}
			if rs, ok := resourceList[corev1.ResourceRequestsStorage]; ok {
				srl.SpaceStorageCapacity = rs.String()
			}
			if rs, ok := resourceList[corev1.ResourceEphemeralStorage]; ok {
				srl.SpaceEphemeralStorage = rs.String()
			}
			if rs, ok := resourceList[corev1.ResourcePersistentVolumeClaims]; ok {
				srl.SpacePvcCount = rs.String()
			}
			if rs, ok := resourceList[corev1.ResourceServicesLoadBalancers]; ok {
				srl.SpaceLbCount = rs.String()
			}
		}
	}

	lrs, err := goClient.GetClientSet().CoreV1().LimitRanges(req.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range lrs.Items {
		ls := item.Spec.Limits
		if len(ls) > 0 {
			if srl == nil {
				srl = &SpaceResourceLimit{}
			}
			for _, l := range ls {
				if l.Type == corev1.LimitTypeContainer {
					limits := l.Default
					if li, ok := limits[corev1.ResourceMemory]; ok {
						srl.ContainerLimitsMem = fmt.Sprintf("%dMi", li.Value()>>20)
					}
					if li, ok := limits[corev1.ResourceCPU]; ok {
						srl.ContainerLimitsCpu = li.String()
					}
					if li, ok := limits[corev1.ResourceEphemeralStorage]; ok {
						srl.ContainerEphemeralStorage = li.String()
					}

					requests := l.DefaultRequest
					if li, ok := requests[corev1.ResourceMemory]; ok {
						srl.ContainerReqMem = fmt.Sprintf("%dMi", li.Value()>>20)
					}
					if li, ok := requests[corev1.ResourceCPU]; ok {
						srl.ContainerReqCpu = li.String()
					}
				}
			}
		}
	}

	if srl != nil {
		resSting, _ := json.Marshal(srl)
		cu.SpaceResourceLimit = string(resSting)
		_, err = service.Svc.ClusterUserSvc.Update(c1, cu)
		if err != nil {
			return err
		}
	}

	for _, i := range req.Collaborator {
		if cu.IsClusterAdmin() {
			if err := cluster_scope.AsCooperator(cu.ClusterId, cu.UserId, i); err != nil {
				errStr = errStr + fmt.Sprintf(",Error while adding %d as cluster cooperator", i)
			}
		} else if err := ns_scope.AsCooperator(cu.ClusterId, i, cu.Namespace); err != nil {
			errStr = errStr + fmt.Sprintf(",Error while adding %d as cluster cooperator", i)
		}
	}

	if errStr != "" {
		errStr = strings.TrimLeft(errStr, ",")
		return errors.New(errStr)
	}
	return nil
}

func NsImport(c *gin.Context) {
	//var cucr ClusterUserCreateRequest
	var req NsImportItem

	if err := c.ShouldBindJSON(&req); err != nil {
		api.SendResponse(c, errno.ErrBind, err.Error())
		return
	}

	resp := struct {
		Success bool
		ErrInfo string
	}{Success: true}

	err := importNs(c, &req)
	if err != nil {
		resp.Success = false
		resp.ErrInfo = err.Error()
	}
	api.SendResponse(c, nil, &resp)
}

type BatchImportItem struct {
	ClusterName  string   `json:"clusterName" yaml:"clusterName"`
	Namespace    string   `json:"namespace" yaml:"namespace"`
	Owner        string   `json:"owner" yaml:"owner"`
	Collaborator []string `json:"collaborator" yaml:"collaborator"`
}

type BatchImport struct {
	Devspaces []*BatchImportItem `json:"devspaces" yaml:"devspaces"`
}

var nsImportStatusMap sync.Map

type NsImportStatusItem struct {
	BatchImportItem
	Success bool
	ErrInfo string `json:"errInfo"`
}

type NsImportStatus struct {
	Process float32
	Items   []*NsImportStatusItem
}

func NsBatchImport(c *gin.Context) {
	//userId, err := ginbase.LoginUser(c)
	//if err != nil {
	//	api.SendResponse(c, errno.ErrPermissionDenied, nil)
	//	return
	//}

	file, err := c.FormFile("upload")
	if err != nil {
		api.SendResponse(c, errno.ErrNsImportFail, err.Error())
		return
	}

	fh, err := file.Open()
	if err != nil {
		api.SendResponse(c, errno.ErrNsImportFail, err.Error())
		return
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(fh)
	if err != nil {
		api.SendResponse(c, errno.ErrNsImportFail, err.Error())
		return
	}
	bys := buf.Bytes()
	batchImport := BatchImport{}
	err = yaml.Unmarshal(bys, &batchImport)
	if err != nil {
		api.SendResponse(c, errno.ErrNsImportFail, err.Error())
		return
	}

	if len(batchImport.Devspaces) == 0 {
		api.SendResponse(c, errno.ErrNsImportFail, "no dev space found")
		return
	}

	uu, _ := uuid.NewUUID()
	task := struct {
		TaskId string `json:"taskId"`
	}{TaskId: uu.String()}

	nsImportStatusMap.Store(task.TaskId, &NsImportStatus{})

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("%v", err)
			}
		}()
		importNsToDevSpace(c, batchImport.Devspaces, task.TaskId)
	}()

	api.SendResponse(c, nil, &task)
	return
}

func ImportStatus(c *gin.Context) {
	taskId := c.Param("id")
	is, ok := nsImportStatusMap.Load(taskId)
	if !ok {
		api.SendResponse(c, errno.ErrUserImport, fmt.Sprintf("task %s not found", taskId))
		return
	}
	api.SendResponse(c, nil, is)
}
