/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package appmeta_manager

import (
	"context"
	"crypto/sha256"
	"encoding"
	"errors"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/resouce_cache"
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

func UpdateApplicationMetasManually(ns string, configBytes []byte, secretName string, secret *v1.Secret) error {
	asw := supervisor.inDeck(ns, configBytes)
	if asw == nil {
		return errors.New("Error while update application manually cause by asw is nil ")
	}
	if secret == nil {
		err := asw.Delete(ns + "/" + secretName)
		log.Infof("receive delete secret operation, name: %s, err: %v", secretName, err)
		return err
	} else {
		err := asw.CreateOrUpdate(ns+"/"+secretName, secret)
		log.Infof("receive update secret operation, name: %s, err: %v", secretName, err)
		return err
	}
}

func GetApplicationMetas(ns string, configBytes []byte) []*appmeta.ApplicationMeta {
	asw := supervisor.inDeck(ns, configBytes)

	if asw == nil {
		return []*appmeta.ApplicationMeta{}
	}
	return asw.GetApplicationMetas()
}

func GetApplicationMeta(ns, appName string, configBytes []byte) *appmeta.ApplicationMeta {
	asw := supervisor.inDeck(ns, configBytes)

	// asw may nil if prepare fail
	meta := asw.GetApplicationMeta(appName, ns)
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
	s.deck[watchDeck] = watcher

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

	searcher, err := resouce_cache.GetSearcherWithLRU(configBytes, ns)
	if err != nil {
		log.ErrorE(err, "Fail to init searcher, and helm watch feature will not be enable")
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

	c, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		log.ErrorE(err, "Fail to init clientSet, and helm watch feature will not be enable")
		return watcher
	}

	clientSet, err := kubernetes.NewForConfig(c)
	if err != nil {
		log.ErrorE(err, "Fail to init clientSet, and helm watch feature will not be enable")
		return watcher
	}

	// we should delete those application installed by helm (still record in secrets)
	// but already deleted
	_ = searcher.Criteria().Namespace(ns).ResourceType("secrets").Consume(
		func(secrets []interface{}) error {
			for _, secret := range secrets {
				v := secret.(*v1.Secret)
				if v.Type == appmeta.SecretType {
					needToDestroy := false

					decode, err := appmeta.Decode(v)
					if err != nil {
						// delete the secret that can not be correctly decode
						log.TLogf(
							"Watcher", "Application Secret '%s' will be deleted, "+
								"the secret is broken.",
							v.Name,
						)

						needToDestroy = true
					} else if _, ok := sets[decode.HelmReleaseName]; !ok && decode.IsInstalled() && decode.ApplicationType.IsHelm() {

						// delete the secret that do not have correspond helm rls
						log.TLogf(
							"Watcher", "Application Secret '%s' will be deleted, "+
								"correspond helm rls is deleted.",
							v.Name,
						)

						needToDestroy = true
					}

					if needToDestroy {
						if err := clientSet.CoreV1().
							Secrets(ns).
							Delete(context.TODO(), v.Name, metav1.DeleteOptions{}); err != nil {
							log.Error(
								err, "Application Secret '%s' from ns %s need to deleted "+
									"but fail.",
								v.Name, ns,
							)
						} else {
							log.TLogf(
								"Watcher", "Application Secret '%s' from ns %s has been be deleted. ",
								v.Name, ns,
							)
						}
					}
				}
			}
			return nil
		},
	)

	go func() {
		helmSecretWatcher.Watch()
	}()
	go func() {
		helmConfigmapWatcher.Watch()
	}()

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
