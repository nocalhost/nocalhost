/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package setupcluster

import (
	"sync"

	"nocalhost/pkg/nocalhost-api/pkg/clientgo"
	"nocalhost/pkg/nocalhost-api/pkg/errno"
)

var sharedMeshManagerFactoryCache = NewSharedMeshManagerFactory()

func GetSharedMeshManagerFactory() SharedMeshManagerFactory {
	return sharedMeshManagerFactoryCache
}

type SharedMeshManagerFactory interface {
	Manager(string) (MeshManager, error)
	Check(string) bool
	Delete(string)
}

func NewSharedMeshManagerFactory() SharedMeshManagerFactory {
	return &sharedMeshManagerFactory{
		manager: map[string]MeshManager{},
	}
}

type sharedMeshManagerFactory struct {
	mu      sync.Mutex
	manager map[string]MeshManager
}

func (f *sharedMeshManagerFactory) Manager(kubeconfig string) (MeshManager, error) {
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
	manager = NewMeshManager(client)
	if err := manager.BuildCache(); err != nil {
		return nil, err
	}
	f.manager[kubeconfig] = manager
	return manager, nil
}

func (f *sharedMeshManagerFactory) Delete(kubeconfig string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.manager, kubeconfig)
}

func (f *sharedMeshManagerFactory) Check(kubeconfig string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, exists := f.manager[kubeconfig]
	return exists
}
