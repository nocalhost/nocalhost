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
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/ports"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) NewSyncthing(deployment string, devStartOptions *DevStartOptions, fileSyncOptions *FileSyncOptions) (*syncthing.Syncthing, error) {
	var err error

	// for file-sync, it should directly use ports defined in .profile.yaml
	//remotePort := fileSyncOptions.RemoteSyncthingPort
	//remoteGUIPort := fileSyncOptions.RemoteSyncthingGUIPort
	//localListenPort := fileSyncOptions.LocalSyncthingPort
	//localGuiPort := fileSyncOptions.LocalSyncthingGUIPort
	svcProfile := a.GetSvcProfile(deployment)
	remotePort := svcProfile.RemoteSyncthingPort
	remoteGUIPort := svcProfile.RemoteSyncthingGUIPort
	localListenPort := svcProfile.LocalSyncthingPort
	localGuiPort := svcProfile.LocalSyncthingGUIPort

	// for dev-start(work dir is not nil), it should take some local available port
	//if devStartOptions.WorkDir != "" {
	//	remotePort, err = ports.GetAvailablePort()
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	remoteGUIPort, err = ports.GetAvailablePort()
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	localGuiPort, err = ports.GetAvailablePort()
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	localListenPort, err = ports.GetAvailablePort()
	//	if err != nil {
	//		return nil, err
	//	}
	//}

	if remotePort == 0 {
		remotePort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
	}

	if remoteGUIPort == 0 {
		remoteGUIPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
	}

	if localGuiPort == 0 {
		localGuiPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
	}

	if localListenPort == 0 {
		localListenPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(syncthing.Nocalhost), 0)
	if err != nil {
		log.Debugf("couldn't hash the password %s", err)
		hash = []byte("")
	}
	sendMode := syncthing.DefaultSyncMode
	if !fileSyncOptions.SyncDouble {
		sendMode = syncthing.SendOnlySyncMode
	}
	s := &syncthing.Syncthing{
		APIKey:           "nocalhost",
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

		// from nhctl config edit
		//SyncedPattern:  fileSyncOptions.SyncedPattern,
		//IgnoredPattern: fileSyncOptions.IgnoredPattern,
		SyncedPattern:  svcProfile.SyncedPattern,
		IgnoredPattern: svcProfile.IgnoredPattern,
	}

	// TODO, warn: multi local sync dir is Deprecated, now it's implement by IgnoreFiles
	// before creating syncthing sidecar, it need to know how many directories it should sync
	index := 1
	for _, sync := range devStartOptions.LocalSyncDir {
		result, err := syncthing.IsSubPathFolder(sync, devStartOptions.LocalSyncDir)
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
					RemotePath: a.GetDefaultWorkDir(deployment),
				},
			)
			index++
		}
	}

	return s, nil
}

func (a *Application) CreateSyncThingSecret(svcName string, syncSecret *corev1.Secret) error {

	// check if secret exist
	exist, err := a.client.GetSecret(context.TODO(), a.GetNamespace(), syncSecret.Name)
	if exist.Name != "" {
		_ = a.client.DeleteSecret(context.TODO(), a.GetNamespace(), syncSecret.Name)
	}
	sc, err := a.client.CreateSecret(context.TODO(), a.GetNamespace(), syncSecret, metav1.CreateOptions{})
	if err != nil {
		// TODO check configmap first, and end dev should delete that secret
		return err
		//log.Fatalf("create syncthing secret fail, please try to manual delete %s secret first", syncthing.SyncSecretName)
	}

	svcPro := a.GetSvcProfile(svcName)
	svcPro.SyncthingSecret = sc.Name
	return a.AppProfile.Save()
}
