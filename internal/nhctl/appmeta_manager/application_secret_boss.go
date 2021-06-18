/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package appmeta_manager

import (
	"crypto/sha256"
	"encoding"
	"fmt"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

var (
	supervisor = &Supervisor{
		deck: map[string]*applicationSecretWatcher{},
	}
)

type Supervisor struct {
	deck map[string]*applicationSecretWatcher
	lock sync.Mutex
}

func GetApplicationMetas(ns string, configBytes []byte) []*appmeta.ApplicationMeta {
	aws := supervisor.inDeck(ns, configBytes)
	return aws.GetApplicationMetas()
}

func GetApplicationMeta(ns, appName string, configBytes []byte) *appmeta.ApplicationMeta {
	aws := supervisor.inDeck(ns, configBytes)

	// aws may nil if prepare fail
	meta := aws.GetApplicationMeta(appName, ns)
	return meta
}

func (s *Supervisor) getIDeck(ns string, configBytes []byte) *applicationSecretWatcher {
	if asw, ok := s.deck[s.key(ns, configBytes)]; ok {
		return asw
	}
	return nil
}

func (s *Supervisor) inDeck(ns string, configBytes []byte) *applicationSecretWatcher {
	if asw := s.getIDeck(ns, configBytes); asw != nil {
		return asw
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// double check
	if asw := s.getIDeck(ns, configBytes); asw != nil {
		return asw
	}
	watchDeck := s.key(ns, configBytes)

	watcher := NewApplicationSecretWatcher(configBytes, ns)

	log.Infof("Prepare SecretWatcher for ns %s", ns)
	if err := watcher.Prepare(); err != nil {
		log.ErrorE(err, "Error while prepare watcher for ns "+ns)
		return nil
	}

	log.Infof("Prepare complete, start to watch for ns %s", ns)
	go func() {
		watcher.Watch()
		s.outDeck(ns, configBytes)
	}()

	helmSecretWatcher := NewHelmSecretWatcher(configBytes,ns)
	log.Infof("Prepare HelmSecretWatcher for ns %s", ns)
	if err := helmSecretWatcher.Prepare(); err == nil {
		log.Infof("Prepare complete, start to watch helm secret for ns %s", ns)
		go func() {
			helmSecretWatcher.Watch()
		}()
	}

	helmConfigmapWatcher := NewHelmCmWatcher(configBytes,ns)
	log.Infof("Prepare HelmCmWatcher for ns %s", ns)
	if err := helmConfigmapWatcher.Prepare(); err == nil {
		log.Infof("Prepare complete, start to watch helm cm for ns %s", ns)
		go func() {
			helmConfigmapWatcher.Watch()
		}()
	}

	s.deck[watchDeck] = watcher
	return watcher
}

func (s *Supervisor) outDeck(ns string, configBytes []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.deck, s.key(ns, configBytes))
}

func (s *Supervisor) key(ns string, configBytes []byte) string {
	sha := sha256.New()
	sha.Write(configBytes)

	marshaler, ok := sha.(encoding.BinaryMarshaler)
	if !ok {
		log.Fatal("first does not implement encoding.BinaryMarshaler")
	}

	state, err := marshaler.MarshalBinary()
	if err != nil {
		log.Fatal("unable to marshal hash:", err)
	}

	return fmt.Sprintf("%s[%s]", ns, string(state))
}
