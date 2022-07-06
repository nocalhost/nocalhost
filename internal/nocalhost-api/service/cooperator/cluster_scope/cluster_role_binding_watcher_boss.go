/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster_scope

import (
	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

var (
	supervisor = &Supervisor{
		deck: sync.Map{},
	}
)

func IsCooperOfCluster(clusterId uint64, saName string) bool {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return false
	}

	deck := supervisor.getInDeck(cc.KubeConfig)

	var isCooperOfCluster = false

	// map[sa]*saShareContainer
	deck.cache.Range(
		func(sa, ssc interface{}) bool {
			if (ssc.(*saShareContainer)).cooperatorSas.Exist(saName) {
				isCooperOfCluster = true
				return false
			}
			return true
		},
	)
	return isCooperOfCluster
}

func IsViewerOfCluster(clusterId uint64, saName string) bool {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return false
	}

	deck := supervisor.getInDeck(cc.KubeConfig)

	var isViewerOfCluster = false

	// map[sa]*saShareContainer
	deck.cache.Range(
		func(sa, ssc interface{}) bool {
			if (ssc.(*saShareContainer)).viewerSas.Exist(saName) {
				isViewerOfCluster = true
				return false
			}
			return true
		},
	)
	return isViewerOfCluster
}

func IsOwnerOfCluster(clusterId uint64, saName string) bool {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return false
	}

	deck := supervisor.getInDeck(cc.KubeConfig)
	load, ok := deck.cache.Load(saName)
	if !ok {
		return false
	}

	return load.(*saShareContainer).ownerValid
}

func IsValidOwner(clusterId, fromUserId uint64) bool {
	ssc := getShareContainer(clusterId, fromUserId)

	if ssc == nil {
		return false
	}
	return ssc.ownerValid
}

func IsCooperAs(clusterId, fromUserId, shareUserId uint64) bool {
	uc, err := service.Svc.UserSvc.GetCache(shareUserId)
	if err != nil || uc.SaName == "" {
		return false
	}

	ssc := getShareContainer(clusterId, fromUserId)

	if ssc == nil {
		return false
	}
	return ssc.cooperatorSas.Exist(uc.SaName)
}

func IsViewerAs(clusterId, fromUserId, shareUserId uint64) bool {
	uc, err := service.Svc.UserSvc.GetCache(shareUserId)
	if err != nil || uc.SaName == "" {
		return false
	}

	ssc := getShareContainer(clusterId, fromUserId)

	if ssc == nil {
		return false
	}
	return ssc.viewerSas.Exist(uc.SaName)
}

func CoopSas(clusterId, fromUserId uint64) []string {
	ssc := getShareContainer(clusterId, fromUserId)

	if ssc == nil {
		return nil
	}
	return ssc.cooperatorSas.ToArray()
}

func ViewSas(clusterId, fromUserId uint64) []string {
	ssc := getShareContainer(clusterId, fromUserId)

	if ssc == nil {
		return nil
	}
	return ssc.viewerSas.ToArray()
}

func getShareContainer(clusterId, fromUserId uint64) *saShareContainer {
	cc, uc, err := GetFromCache(clusterId, fromUserId)
	if err != nil {
		return nil
	}

	deck := supervisor.getInDeck(cc.KubeConfig)

	// map[sa]*saShareContainer
	load, ok := deck.cache.Load(uc.SaName)
	if !ok {
		return nil
	}
	return load.(*saShareContainer)
}

func GetFromCache(clusterId, fromUserId uint64) (
	*model.ClusterModel, *model.UserBaseModel, error) {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return nil, nil, err
	}

	uc, err := service.Svc.UserSvc.GetCache(fromUserId)
	if err != nil {
		return nil, nil, err
	}

	return &cc, &uc, nil
}

type Supervisor struct {
	deck sync.Map // map[string]*clusterRoleBindingWatcher{}
}

func (s *Supervisor) getInDeck(kubeConfig string) *clusterRoleBindingWatcher {
	// double check
	if asw := s.inDeck(kubeConfig); asw != nil {
		return asw
	}

	watcher := NewClusterRoleBindingWatcher(kubeConfig)

	if err := watcher.Prepare(); err != nil {
		log.ErrorE(err, "Error while prepare role binding watcher")
		return watcher
	}

	go func() {
		watcher.Watch()
		s.outDeck(kubeConfig)
	}()

	s.deck.Store(kubeConfig, watcher)
	return watcher
}

func (s *Supervisor) inDeck(kubeConfig string) *clusterRoleBindingWatcher {
	if asw, ok := s.deck.Load(kubeConfig); ok {
		return asw.(*clusterRoleBindingWatcher)
	}
	return nil
}

func (s *Supervisor) outDeck(kubeConfig string) {
	s.deck.Delete(kubeConfig)
}
