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
	"nocalhost/internal/nhctl/nocalhost"
	"path/filepath"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/log"
)

func (a *Application) NewSyncthing(deployment string, devStartOptions *DevStartOptions, fileSyncOptions *FileSyncOptions) (*syncthing.Syncthing, error) {
	var err error

	// for file-sync, it should directly use ports defined in .profile.yaml
	remotePort := fileSyncOptions.RemoteSyncthingPort
	remoteGUIPort := fileSyncOptions.RemoteSyncthingGUIPort
	localListenPort := fileSyncOptions.LocalSyncthingPort
	localGuiPort := fileSyncOptions.LocalSyncthingGUIPort

	// for dev-start(work dir is not nil), it should take some local available port
	if devStartOptions.WorkDir != "" {
		remotePort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}

		remoteGUIPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}

		localGuiPort, err = ports.GetAvailablePort()
		if err != nil {
			return nil, err
		}

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
	}

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
					RemotePath: devStartOptions.WorkDir,
				},
			)
			index++
		}
	}

	return s, nil
}
