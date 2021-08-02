/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package service_account

import (
	"nocalhost/pkg/nhctl/log"
	"sort"
	"sync"
)

var (
	supervisor = &Supervisor{
		deck: map[string]*roleBindingWatcher{},
	}
)

type Supervisor struct {
	deck map[string]*roleBindingWatcher
	lock sync.Mutex
}

func GetAllPermittedNs(kubeConfig string, saName string) []string {
	watcher := supervisor.getInDeck(kubeConfig)

	var result []string
	if watcher == nil {
		return result
	}

	ownNs := watcher.ownNs

	for namespace, set := range ownNs {
		if set.exist(saName) {
			result = append(result, namespace)
		}
	}

	sort.Strings(result)
	return result
}

func (s *Supervisor) getInDeck(kubeConfig string) *roleBindingWatcher {
	if asw := s.inDeck(kubeConfig); asw != nil {
		return asw
	}

	s.lock.Lock()
	defer s.lock.Unlock()

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

	s.deck[kubeConfig] = watcher
	return watcher
}

func (s *Supervisor) inDeck(kubeConfig string) *roleBindingWatcher {
	if asw, ok := s.deck[kubeConfig]; ok {
		return asw
	}
	return nil
}

func (s *Supervisor) outDeck(kubeConfig string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.deck, kubeConfig)
}
