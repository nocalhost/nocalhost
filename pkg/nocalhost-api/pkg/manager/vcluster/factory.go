/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package vcluster

import (
	"sync"

	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

var sharedManagerFactoryCache = NewSharedManagerFactory()

func GetSharedManagerFactory() SharedManagerFactory {
	return sharedManagerFactoryCache
}

type SharedManagerFactory interface {
	Manager(string) (Manager, error)
	Delete(string)
}

func NewSharedManagerFactory() SharedManagerFactory {
	return &sharedManagerFactory{
		manager: map[string]Manager{},
	}
}

type sharedManagerFactory struct {
	mu      sync.Mutex
	manager map[string]Manager
}

func (f *sharedManagerFactory) Manager(kubeconfig string) (Manager, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	manager, exists := f.manager[kubeconfig]
	if exists {
		return manager, nil
	}
	client, err := clientgo.NewAdminGoClient([]byte(kubeconfig))
	if err != nil {
		switch err.(type) {
		case *errno.Errno:
			return nil, err
		default:
			return nil, errno.ErrClusterKubeErr
		}
	}
	manager = newManager(client)
	f.manager[kubeconfig] = manager
	return manager, nil
}

func (f *sharedManagerFactory) Delete(kubeconfig string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, exists := f.manager[kubeconfig]
	if !exists {
		return
	}
	m.close()
	delete(f.manager, kubeconfig)
}
