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
