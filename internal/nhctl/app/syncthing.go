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

package app

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/internal/nhctl/syncthing/ports"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) NewSyncthing(deployment string, container string, localSyncDir []string, syncDouble bool) (*syncthing.Syncthing, error) {
	var err error

	svcProfile := a.GetSvcProfileV2(deployment)
	remotePort := svcProfile.RemoteSyncthingPort
	remoteGUIPort := svcProfile.RemoteSyncthingGUIPort
	localListenPort := svcProfile.LocalSyncthingPort
	localGuiPort := svcProfile.LocalSyncthingGUIPort

	if remotePort == 0 {
		remotePort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
		svcProfile.RemoteSyncthingPort = remotePort
	}

	if remoteGUIPort == 0 {
		remoteGUIPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
		svcProfile.RemoteSyncthingGUIPort = remoteGUIPort
	}

	if localGuiPort == 0 {
		localGuiPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
		svcProfile.LocalSyncthingGUIPort = localGuiPort
	}

	if localListenPort == 0 {
		localListenPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
		svcProfile.LocalSyncthingPort = localListenPort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(syncthing.Nocalhost), 0)
	if err != nil {
		log.Debugf("couldn't hash the password %s", err)
		hash = []byte("")
	}
	sendMode := syncthing.DefaultSyncMode
	if !syncDouble {
		sendMode = syncthing.SendOnlySyncMode
	}
	s := &syncthing.Syncthing{
		APIKey:           syncthing.DefaultAPIKey,
		GUIPassword:      "nocalhost",
		GUIPasswordHash:  string(hash),
		BinPath:          filepath.Join(nocalhost.GetSyncThingBinDir(), syncthing.GetBinaryName()),
		Client:           syncthing.NewAPIClient(),
		FileWatcherDelay: syncthing.DefaultFileWatcherDelay,
		GUIAddress:       fmt.Sprintf("%s:%d", syncthing.Bind, localGuiPort),
		// TODO Be Careful if ResourcePath is not application path, Local syncthing HOME PATH will be used for cert and config.xml
		// it's `~/.nhctl/application/bookinfo/syncthing`
		LocalHome:        filepath.Join(a.GetHomeDir(), "syncthing", deployment),
		RemoteHome:       syncthing.RemoteHome,
		LogPath:          filepath.Join(a.GetHomeDir(), "syncthing", deployment, syncthing.LogFile),
		RemoteAddress:    fmt.Sprintf("%s:%d", syncthing.Bind, remotePort),
		RemoteDeviceID:   syncthing.DefaultRemoteDeviceID,
		RemoteGUIAddress: fmt.Sprintf("%s:%d", syncthing.Bind, remoteGUIPort),
		RemoteGUIPort:    remoteGUIPort,
		RemotePort:       remotePort,
		LocalGUIPort:     localGuiPort,
		LocalPort:        localListenPort,
		ListenAddress:    fmt.Sprintf("%s:%d", syncthing.Bind, localListenPort),
		Type:             sendMode, // sendonly mode
		IgnoreDelete:     true,
		Folders:          []*syncthing.Folder{},
		RescanInterval:   "300",

		//SyncedPattern:  svcProfile.GetDefaultContainerDevConfig().Sync.FilePattern,
		//IgnoredPattern: svcProfile.GetDefaultContainerDevConfig().Sync.IgnoreFilePattern,
	}
	if svcProfile.GetContainerDevConfigOrDefault(container).Sync != nil {
		s.SyncedPattern = svcProfile.GetContainerDevConfigOrDefault(container).Sync.FilePattern
		s.IgnoredPattern = svcProfile.GetContainerDevConfigOrDefault(container).Sync.IgnoreFilePattern
	}

	// TODO, warn: multi local sync dir is Deprecated, now it's implement by IgnoreFiles
	// before creating syncthing sidecar, it need to know how many directories it should sync
	index := 1
	for _, sync := range localSyncDir {
		result, err := syncthing.IsSubPathFolder(sync, localSyncDir)
		// TODO considering continue on err
		if err != nil {
			return nil, err
		}
		if !result {
			s.Folders = append(
				s.Folders,
				&syncthing.Folder{
					Name:       strconv.Itoa(index),
					LocalPath:  sync,
					RemotePath: a.GetDefaultWorkDir(deployment, container),
				},
			)
			index++
		}
	}

	_ = a.SaveProfile()

	return s, nil
}

func (a *Application) NewSyncthingHttpClient(svcName string) *req.SyncthingHttpClient {
	svcProfile := a.GetSvcProfileV2(svcName)

	return req.NewSyncthingHttpClient(
		fmt.Sprintf("127.0.0.1:%d", svcProfile.LocalSyncthingGUIPort),
		syncthing.DefaultAPIKey,
		syncthing.DefaultRemoteDeviceID,
		syncthing.DefaultFolderName,
	)
}

func (a *Application) CreateSyncThingSecret(svcName string, syncSecret *corev1.Secret) error {

	// check if secret exist
	exist, err := a.client.GetSecret(syncSecret.Name)
	if exist.Name != "" {
		_ = a.client.DeleteSecret(syncSecret.Name)
	}
	sc, err := a.client.CreateSecret(syncSecret)
	if err != nil {
		return err
	}

	svcPro := a.GetSvcProfileV2(svcName)
	svcPro.SyncthingSecret = sc.Name
	return a.SaveProfile()
}
