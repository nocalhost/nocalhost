/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"fmt"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/internal/nhctl/syncthing/ports"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"os"
	"path/filepath"
	"strconv"

	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/pkg/nhctl/log"
)

func (c *Controller) NewSyncthing(container string, localSyncDir []string, syncDouble bool) (
	*syncthing.Syncthing, error,
) {
	var err error
	remotePath := c.GetWorkDir(container)
	appProfile, err := c.GetProfileForUpdate()
	if err != nil {
		return nil, err
	}
	defer func() {
		if appProfile != nil {
			_ = appProfile.CloseDb()
		}
	}()
	svcProfile := appProfile.SvcProfileV2(c.Name, c.Type.String())
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

	sendMode := _const.DefaultSyncType
	if !syncDouble {
		sendMode = _const.SendOnlySyncType
	}
	localHomeDir := c.GetSyncDir()

	s := &syncthing.Syncthing{
		APIKey:           syncthing.DefaultAPIKey,
		BinPath:          filepath.Join(nocalhost.GetSyncThingBinDir(), syncthing.GetBinaryName()),
		Client:           syncthing.NewAPIClient(),
		FileWatcherDelay: syncthing.DefaultFileWatcherDelay,
		GUIAddress:       fmt.Sprintf("%s:%d", syncthing.Bind, localGuiPort),
		// TODO Be Careful if ResourcePath is not application path, Local
		// syncthing HOME PATH will be used for cert and config.xml
		LocalHome:        localHomeDir,
		RemoteHome:       syncthing.RemoteHome,
		LogPath:          filepath.Join(localHomeDir, syncthing.LogFile),
		RemoteAddress:    fmt.Sprintf("%s:%d", syncthing.Bind, remotePort),
		RemoteDeviceID:   syncthing.DefaultRemoteDeviceID,
		RemoteGUIAddress: fmt.Sprintf("%s:%d", syncthing.Bind, remoteGUIPort),
		RemoteGUIPort:    remoteGUIPort,
		RemotePort:       remotePort,
		LocalGUIPort:     localGuiPort,
		LocalPort:        localListenPort,
		ListenAddress:    fmt.Sprintf("%s:%d", syncthing.Bind, localListenPort),
		Type:             sendMode, // sendonly mode
		Folders:          []*syncthing.Folder{},
		RescanInterval:   "300",
	}
	svcConfig := c.Config()
	devConfig := svcConfig.GetContainerDevConfigOrDefault(container)
	if devConfig != nil && devConfig.Sync != nil {
		// enable delete protection by default
		// or use the val user specify
		s.IgnoreDelete = devConfig.Sync.DeleteProtection == nil || *devConfig.Sync.DeleteProtection
		s.EnableParseFromGitIgnore = devConfig.Sync.Mode == _const.GitIgnoreMode
		s.SyncedPattern = devConfig.Sync.FilePattern
		s.IgnoredPattern = devConfig.Sync.IgnoreFilePattern
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
					RemotePath: remotePath,
				},
			)
			index++
		}
	}
	_ = appProfile.Save()
	return s, nil
}

func (c *Controller) NewSyncthingHttpClient(reqTimeoutSecond int) *req.SyncthingHttpClient {
	svcProfile, _ := c.GetProfile()

	return req.NewSyncthingHttpClient(
		fmt.Sprintf("127.0.0.1:%d", svcProfile.LocalSyncthingGUIPort),
		syncthing.DefaultAPIKey,
		syncthing.DefaultRemoteDeviceID,
		syncthing.DefaultFolderName,
		reqTimeoutSecond,
	)
}

func (c *Controller) CreateSyncThingSecret(container string, localSyncDir []string, duplicateDevMode bool) error {

	// Delete service folder
	dir := c.GetSyncDir()
	if err2 := os.RemoveAll(dir); err2 != nil {
		log.Logf("Failed to delete dir: %s before starting syncthing, err: %v", dir, err2)
	}

	newSyncthing, err := c.NewSyncthing(container, localSyncDir, false)
	if err != nil {
		return errors.Wrap(err, "Failed to create syncthing process")
	}
	// set syncthing secret
	config, err := newSyncthing.GetRemoteConfigXML()
	if err != nil {
		return err
	}

	syncSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.GetSyncThingSecretName(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"config.xml": config,
			"cert.pem":   []byte(secret_config.CertPEM),
			"key.pem":    []byte(secret_config.KeyPEM),
		},
	}

	// check if secret exist
	exist, err := c.Client.GetSecret(syncSecret.Name)
	if exist != nil && exist.Name != "" {
		_ = c.Client.DeleteSecret(syncSecret.Name)
	}
	_, err = c.Client.CreateSecret(syncSecret, metav1.CreateOptions{})
	//if err != nil {
	return err
	//}

	//return c.UpdateSvcProfile(
	//	func(svcPro *profile.SvcProfileV2) error {
	//		if duplicateDevMode {
	//			svcPro.DuplicateDevModeSyncthingSecretName = sc.Name
	//		} else {
	//			svcPro.SyncthingSecret = sc.Name
	//		}
	//		return nil
	//	},
	//)
}
