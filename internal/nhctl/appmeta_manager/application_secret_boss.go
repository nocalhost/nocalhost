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

func GetApplicationMetas(ns, config string) []*appmeta.ApplicationMeta {
	aws := supervisor.inDeck(ns, config)
	return aws.GetApplicationMetas()
}

func GetApplicationMeta(ns, appName, config string) *appmeta.ApplicationMeta {
	aws := supervisor.inDeck(ns, config)
	meta := aws.GetApplicationMeta(appName)
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
	go func() {
		watcher.Watch()
		s.outDeck(ns, config)
	}()
	s.deck[watchDeck] = watcher
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
