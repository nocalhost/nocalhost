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

package service_account

import (
	"nocalhost/pkg/nhctl/log"
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
