/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package nocalhost

import (
	"encoding/json"
	"io/ioutil"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"nocalhost/internal/nhctl/utils"
)

const (
	DefaultNewFilePermission        = 0700
	DefaultApplicationDirName       = "application"
	DefaultBinDirName               = "bin"
	DefaultBinSyncThingDirName      = "syncthing"
	DefaultLogDirName               = "logs"
	DefaultLogFileName              = "nhctl.log"
	DefaultApplicationProfilePath   = ".profile.yaml" // runtime config
	DefaultApplicationProfileV2Path = ".profile_v2.yaml"
)

func Init() error {
	var err error
	nhctlHomeDir := nocalhost_path.GetNhctlHomeDir()
	if _, err = os.Stat(nhctlHomeDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(nhctlHomeDir, DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			// Initial ns dir
			nsDir := nocalhost_path.GetNhctlNameSpaceDir()
			err = os.MkdirAll(nsDir, DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			binDir := GetSyncThingBinDir()
			err = os.MkdirAll(binDir, DefaultNewFilePermission) // create .nhctl/bin/syncthing
			if err != nil {
				return errors.Wrap(err, "")
			}

			logDir := GetLogDir()
			err = os.MkdirAll(logDir, DefaultNewFilePermission) // create .nhctl/logs
			if err != nil {
				return errors.Wrap(err, "")
			}

		}
	}
	return err
}

func CleanupAppFilesUnderNs(appName string, namespace string) error {
	appDir := nocalhost_path.GetAppDirUnderNs(appName, namespace)
	if f, err := os.Stat(appDir); err == nil {
		if f.IsDir() {
			err = os.RemoveAll(appDir)
			return errors.Wrap(err, "fail to remove dir")
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "")
	}
	return nil
}

func GetSyncThingBinDir() string {
	return filepath.Join(nocalhost_path.GetNhctlHomeDir(), DefaultBinDirName, DefaultBinSyncThingDirName)
}

func GetLogDir() string {
	return filepath.Join(nocalhost_path.GetNhctlHomeDir(), DefaultLogDirName)
}

// key: ns, value: app
// Deprecated
func GetNsAndApplicationInfo() (map[string][]string, error) {
	result := make(map[string][]string, 0)
	nsDir := nocalhost_path.GetNhctlNameSpaceDir()
	nsList, err := ioutil.ReadDir(nsDir)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	for _, ns := range nsList {
		appList := make([]string, 0)
		if !ns.IsDir() {
			continue
		}
		appDirList, err := ioutil.ReadDir(filepath.Join(nsDir, ns.Name()))
		if err != nil {
			log.WarnE(errors.Wrap(err, ""), "Failed to read dir")
			continue
		}
		for _, appDir := range appDirList {
			if appDir.IsDir() {
				appList = append(appList, appDir.Name())
			}
		}
		result[ns.Name()] = appList
	}
	return result, nil
}

func GetApplicationMeta(appName, namespace, kubeConfig string) (*appmeta.ApplicationMeta, error) {
	cli, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
	if err != nil {
		return nil, err
	}

	bys, err := ioutil.ReadFile(kubeConfig)

	if err != nil {
		return nil, errors.Wrap(err, "Error to get ApplicationMeta while read kubeconfig")
	}

	data, err := cli.SendGetApplicationMetaCommand(namespace, appName, string(bys))
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	marshal, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	appMeta := &appmeta.ApplicationMeta{}
	err = json.Unmarshal(marshal, appMeta)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	// applicationMeta use the kubeConfig content, but there use the path to init client
	// unexpect error occur if someone change the content of kubeConfig before InitGoClient
	// and after SendGetApplicationMetaCommand
	if err = appMeta.InitGoClient(kubeConfig); err != nil {
		return nil, err
	}

	return appMeta, nil
}

func GetApplicationMetas(namespace, kubeConfig string) (appmeta.ApplicationMetas, error) {
	cli, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
	if err != nil {
		return nil, err
	}

	bys, err := ioutil.ReadFile(kubeConfig)

	if err != nil {
		return nil, errors.Wrap(err, "Error to get ApplicationMeta")
	}

	data, err := cli.SendGetApplicationMetasCommand(namespace, string(bys))
	if err != nil {
		return nil, err
	}
	var appMetas []*appmeta.ApplicationMeta
	if data == nil {
		return appMetas, nil
	}
	marshal, err := json.Marshal(data)
	err = json.Unmarshal(marshal, &appMetas)
	if err != nil {
		return nil, err
	}
	// applicationMeta use the kubeConfig content, but there use the path to init client
	// unexpect error occur if someone change the content of kubeConfig before InitGoClient
	// and after SendGetApplicationMetaCommand
	for _, meta := range appMetas {
		if err = meta.InitGoClient(kubeConfig); err != nil {
			return nil, err
		}
	}

	return appMetas, nil
}
