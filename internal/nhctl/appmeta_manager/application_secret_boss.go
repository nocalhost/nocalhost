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

package appmeta_manager

import (
	"crypto/sha256"
	"encoding"
	"encoding/json"
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

func GetApplicationMetas(ns, config string) []*appmeta.ApplicationMeta {
	aws := supervisor.inDeck(ns, config)
	return aws.GetApplicationMetas()
}

func GetApplicationMeta(ns, appName, config string) *appmeta.ApplicationMeta {
	aws := supervisor.inDeck(ns, config)

	// aws may nil if prepare fail
	meta := aws.GetApplicationMeta(appName, ns)
	return meta
}

func (s *Supervisor) getIDeck(ns, config string) *applicationSecretWatcher {
	if asw, ok := s.deck[s.key(ns, config)]; ok {
		return asw
	}
	return nil
}

func (s *Supervisor) inDeck(ns, config string) *applicationSecretWatcher {
	if asw := s.getIDeck(ns, config); asw != nil {
		return asw
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// double check
	if asw := s.getIDeck(ns, config); asw != nil {
		return asw
	}
	watchDeck := s.key(ns, config)

	watcher := NewApplicationSecretWatcher(config, ns)

	log.Infof("Prepare for ns %s", ns)
	if err := watcher.Prepare(); err != nil {
		log.ErrorE(err, "Error while prepare watcher for ns "+ns)
		return nil
	}

	log.Infof("Prepare complete, start to watch for ns %s", ns)
	go func() {
		watcher.Watch()
		s.outDeck(ns, config)
	}()

	s.deck[watchDeck] = watcher

	marshal, _ := json.Marshal(watcher.applicationMetas)
	log.Infof("applicationMetas:   %s", marshal)

	return watcher
}

func (s *Supervisor) outDeck(ns, config string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.deck, s.key(ns, config))
}

func (s *Supervisor) key(ns, kubeConfig string) string {
	sha := sha256.New()
	sha.Write([]byte(kubeConfig))

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
