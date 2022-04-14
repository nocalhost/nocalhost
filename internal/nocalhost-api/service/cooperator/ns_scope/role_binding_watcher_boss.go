/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ns_scope

import (
	"nocalhost/internal/nocalhost-api/service"
	"nocalhost/pkg/nhctl/log"
	"sort"
	"sync"
)

var (
	supervisor = &Supervisor{
		deck: sync.Map{},
	}
)

type Supervisor struct {
	deck sync.Map // map[string]*roleBindingWatcher
}

func CoopSas(clusterId uint64, namespace string) []string {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return nil
	}

	kubeConfig := cc.KubeConfig
	watcher := supervisor.getInDeck(kubeConfig)
	if watcher == nil {
		return nil
	}

	if set, ok := watcher.cooperatorSas[namespace]; ok {
		return set.toArray()
	}
	return nil
}

func ViewerSas(clusterId uint64, namespace string) []string {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return nil
	}

	kubeConfig := cc.KubeConfig
	watcher := supervisor.getInDeck(kubeConfig)
	if watcher == nil {
		return nil
	}

	if set, ok := watcher.viewerSas[namespace]; ok {
		return set.toArray()
	}
	return nil
}

func AllOwnNs(clusterId, userId uint64) []string {
	if kubeconfig, saName, err := getFromCache(clusterId, userId); err != nil {
		return nil
	} else {
		return GetAllOwnNs(kubeconfig, saName)
	}
}

func AllCoopNs(clusterId, userId uint64) []string {
	if kubeconfig, saName, err := getFromCache(clusterId, userId); err != nil {
		return nil
	} else {
		return GetAllCoopNs(kubeconfig, saName)
	}
}

func AllViewNs(clusterId, userId uint64) []string {
	if kubeconfig, saName, err := getFromCache(clusterId, userId); err != nil {
		return nil
	} else {
		return GetAllViewNs(kubeconfig, saName)
	}
}

func getFromCache(clusterId, userId uint64) (
	kubeConfig string, saName string, err error,
) {
	cc, err := service.Svc.ClusterSvc.GetCache(clusterId)
	if err != nil {
		return
	}
	kubeConfig = cc.KubeConfig

	uc, err := service.Svc.UserSvc.GetCache(userId)
	if err != nil {
		return
	}
	saName = uc.SaName

	return
}

func GetAllOwnNs(kubeConfig string, saName string) []string {
	watcher := supervisor.getInDeck(kubeConfig)

	var result []string
	if watcher == nil {
		return result
	}

	watcher.lock.RLock()
	defer watcher.lock.RUnlock()

	ownNs := watcher.adminSas

	for namespace, set := range ownNs {
		if set.exist(saName) {
			result = append(result, namespace)
		}
	}

	sort.Strings(result)
	return result
}

func GetAllCoopNs(kubeConfig string, saName string) []string {
	watcher := supervisor.getInDeck(kubeConfig)

	var result []string
	if watcher == nil {
		return result
	}

	watcher.lock.RLock()
	defer watcher.lock.RUnlock()

	cooperNs := watcher.cooperatorSas

	for namespace, set := range cooperNs {
		if set.exist(saName) {
			result = append(result, namespace)
		}
	}

	sort.Strings(result)
	return result
}

func GetAllViewNs(kubeConfig string, saName string) []string {
	watcher := supervisor.getInDeck(kubeConfig)

	var result []string
	if watcher == nil {
		return result
	}

	watcher.lock.RLock()
	defer watcher.lock.RUnlock()

	viewNs := watcher.viewerSas

	for namespace, set := range viewNs {
		if set.exist(saName) {
			result = append(result, namespace)
		}
	}

	sort.Strings(result)
	return result
}

func (s *Supervisor) getInDeck(kubeConfig string) *roleBindingWatcher {
	// double check
	if asw := s.inDeck(kubeConfig); asw != nil {
		return asw
	}

	watcher := NewRoleBindingWatcher(kubeConfig)

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

func (s *Supervisor) inDeck(kubeConfig string) *roleBindingWatcher {
	if asw, ok := s.deck.Load(kubeConfig); ok {
		return asw.(*roleBindingWatcher)
	}
	return nil
}

func (s *Supervisor) outDeck(kubeConfig string) {
	s.deck.Delete(kubeConfig)
}
