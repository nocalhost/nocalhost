package app

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/ports"
	"nocalhost/pkg/nhctl/log"
	"path/filepath"
	"strconv"
)

func (a *Application) NewSyncthing(deployment string, devStartOptions *DevStartOptions, fileSyncOptions *FileSyncOptions) (*syncthing.Syncthing, error) {
	var err error

	// 2. from file-sync it should direct use .profile.yaml dev-start port
	// WorkDir will be null
	remotePort := fileSyncOptions.RemoteSyncthingPort
	remoteGUIPort := fileSyncOptions.RemoteSyncthingGUIPort
	localListenPort := fileSyncOptions.LocalSyncthingPort
	localGuiPort := fileSyncOptions.LocalSyncthingGUIPort

	// 1. from dev-start should take some local available port, WorkDir will be not null
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
		BinPath:          filepath.Join(a.GetSyncThingBinDir(), syncthing.GetBinaryName()),
		Client:           syncthing.NewAPIClient(),
		FileWatcherDelay: syncthing.DefaultFileWatcherDelay,
		GUIAddress:       fmt.Sprintf("%s:%d", syncthing.Bind, localGuiPort),
		// TODO BE CAREFUL ResourcePath if is not application path, this is Local syncthing HOME PATH, use for cert and config.xml
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

	// when create syncthing sidecar it need to know how many directory it should sync
	index := 1
	for _, sync := range devStartOptions.LocalSyncDir {
		result, err := syncthing.IsSubPathFolder(sync, devStartOptions.LocalSyncDir)
		// TODO consider continue handle next dir
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
