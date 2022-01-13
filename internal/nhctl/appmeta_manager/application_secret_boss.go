/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package appmeta_manager

import (
	"crypto/sha256"
	"encoding"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/appmeta"
	profile2 "nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/resouce_cache"
	"nocalhost/pkg/nhctl/log"
	"sync"
)

var (
	supervisor = &Supervisor{
		deck: sync.Map{},
	}
)

type Supervisor struct {
	// deck map[string]*applicationSecretWatcher
	deck sync.Map
	// lock map[string]*sync.lock
	lock sync.Map
}

func UpdateApplicationMetasManually(ns string, configBytes []byte, secretName string, secret *v1.Secret) error {
	asw := supervisor.inDeck(ns, configBytes)
	if asw == nil {
		return errors.New("Error while update application manually cause by asw is nil ")
	}
	if secret == nil {
		err := asw.Delete(ns + "/" + secretName)
		log.Debugf("receive delete secret operation, name: %s, err: %v", secretName, err)
		return err
	} else {
		err := asw.CreateOrUpdate(ns+"/"+secretName, secret)
		log.Debugf("receive update secret operation, name: %s, err: %v", secretName, err)
		return err
	}
}

func GetApplicationMetas(ns string, configBytes []byte, filter func(*appmeta.ApplicationMeta) bool) []*appmeta.ApplicationMeta {
	asw := supervisor.inDeck(ns, configBytes)

	if asw == nil {
		return []*appmeta.ApplicationMeta{}
	}

	valid := make([]*appmeta.ApplicationMeta, 0)
	metas := asw.GetApplicationMetas()

	for _, meta := range metas {
		if filter(meta) {
			valid = append(valid, meta)
		}
	}
	return valid
}

func GetApplicationMeta(ns, appName string, configBytes []byte) *appmeta.ApplicationMeta {
	asw := supervisor.inDeck(ns, configBytes)

	if asw == nil {
		return &appmeta.ApplicationMeta{
			ApplicationState: appmeta.UNKNOWN,
			Ns:               ns,
			Application:      appName,
			DevMeta:          appmeta.ApplicationDevMeta{},
			Config:           &profile2.NocalHostAppConfigV2{},
		}
	}

	// asw may nil if prepare fail
	meta := asw.GetApplicationMeta(appName)

	if meta == nil {
		meta = &appmeta.ApplicationMeta{
			ApplicationState: appmeta.UNINSTALLED,
			Ns:               ns,
			Application:      appName,
			DevMeta:          appmeta.ApplicationDevMeta{},
			Config:           &profile2.NocalHostAppConfigV2{},
		}
	}

	// try load application from annotations
	if meta.IsNotInstall() {
		appsWithAnnotations := resouce_cache.GetAllAppNameByNamespace(configBytes, ns)
		if appsWithAnnotations.Has(appName) {

			if err := meta.DoWithTempOperator(
				configBytes,
				func() error {

					if err := meta.OneTimesInitial(
						func(meta *appmeta.ApplicationMeta) {
							meta.ApplicationType = appmeta.ManifestLocal
							meta.Ns = ns
							meta.Application = appName
							meta.Config = &profile2.NocalHostAppConfigV2{
								Migrated:          true,
								ApplicationConfig: profile2.ApplicationConfig{},
								ConfigProperties:  profile2.ConfigProperties{},
							}
						}, false,
					); err != nil {
						return err
					}

					return nil
				},
			); err != nil {
				log.TLogf(
					"Watcher", "Initial application '%s' by managed annotations fail, Errors: %s",
					appName, err,
				)
			}
		}
	}

	if meta == nil || meta.Application == "" {
		return &appmeta.ApplicationMeta{
			ApplicationState: appmeta.UNINSTALLED,
			Ns:               ns,
			Application:      appName,
			DevMeta:          appmeta.ApplicationDevMeta{},
			Config:           &profile2.NocalHostAppConfigV2{},
		}
	}

	return meta
}

func (s *Supervisor) getIDeck(ns string, configBytes []byte) *applicationSecretWatcher {
	if loaded, ok := s.deck.Load(s.key(ns, configBytes)); ok {
		if asw, ok := loaded.(*applicationSecretWatcher); ok {
			return asw
		}
	}
	return nil
}

func (s *Supervisor) getLock(ns string, configBytes []byte) *sync.Mutex {
	key := s.key(ns, configBytes)

	store, _ := s.lock.LoadOrStore(key, &sync.Mutex{})
	if asw, ok := store.(*sync.Mutex); ok {
		return asw
	} else {

		// that's cloud not happened
		return &sync.Mutex{}
	}
}

func (s *Supervisor) inDeck(ns string, configBytes []byte) *applicationSecretWatcher {
	lock := s.getLock(ns, configBytes)
	lock.Lock()
	defer lock.Unlock()

	if asw := s.getIDeck(ns, configBytes); asw != nil {
		return asw
	}

	watchDeck := s.key(ns, configBytes)
	watcher := NewApplicationSecretWatcher(configBytes, ns)

	log.Infof("Prepare SecretWatcher for ns %s", ns)
	if err := watcher.Prepare(); err != nil {
		log.TLogf(
			"MetaSecret", "Error while get application in deck from ns %s.. "+
				"return empty array.., Error: %s", ns, err.Error(),
		)
		return nil
	}

	log.Infof("Prepare complete, start to watch for ns %s", ns)
	go func() {
		watcher.Watch()
		s.outDeck(ns, configBytes)
	}()

	if _, hasBeenStored := s.deck.LoadOrStore(watchDeck, watcher); hasBeenStored {
		watcher.Quit()
	}

	helmSecretWatcher := NewHelmSecretWatcher(configBytes, ns)
	log.Infof("Prepare HelmSecretWatcher for ns %s", ns)
	installedSecretRls, err := helmSecretWatcher.Prepare()

	// prevent appmeta watcher initial fail
	if err != nil {
		log.ErrorE(err, "Fail to init helm secret Watcher, and helm watch feature will not be enable")
		return watcher
	}

	helmConfigmapWatcher := NewHelmCmWatcher(configBytes, ns)
	log.Infof("Prepare HelmCmWatcher for ns %s", ns)
	installedCmRls, err := helmConfigmapWatcher.Prepare()
	// prevent appmeta watcher initial fail
	if err != nil {
		log.ErrorE(err, "Fail to init helm cm Watcher, and helm watch feature will not be enable")
		return watcher
	}

	// for o(1)
	sets := make(map[string]interface{})
	for _, rl := range installedSecretRls {
		sets[rl] = ""
	}
	for _, rl := range installedCmRls {
		sets[rl] = ""
	}

	go func() {
		helmSecretWatcher.Watch()
	}()
	go func() {
		helmConfigmapWatcher.Watch()
	}()

	return watcher
}

func (s *Supervisor) outDeck(ns string, configBytes []byte) {
	s.deck.Delete(s.key(ns, configBytes))
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

// GetAllApplicationMetasWithDeepClone get all developing application, will not update appmeta.ApplicationMeta
func GetAllApplicationMetas() []*appmeta.ApplicationMeta {
	if supervisor == nil {
		return nil
	}
	metas := make([]*appmeta.ApplicationMeta, 0)

	supervisor.deck.Range(
		func(key, value interface{}) bool {
			if value != nil {
				if asw, ok := value.(*applicationSecretWatcher); ok {
					metas = append(metas, asw.GetApplicationMetas()...)
				}
			}
			return true
		},
	)
	return metas
}
