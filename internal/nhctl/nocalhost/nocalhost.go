/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"nocalhost/internal/nhctl/appmeta"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"nocalhost/internal/nhctl/utils"
)

func Init() error {
	var err error
	nhctlHomeDir := nocalhost_path.GetNhctlHomeDir()
	if _, err = os.Stat(nhctlHomeDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(nhctlHomeDir, _const.DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			// Initial Ns dir
			nsDir := nocalhost_path.GetNhctlNameSpaceBaseDir()
			err = os.MkdirAll(nsDir, _const.DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			binDir := GetSyncThingBinDir()
			err = os.MkdirAll(binDir, _const.DefaultNewFilePermission) // create .nhctl/bin/syncthing
			if err != nil {
				return errors.Wrap(err, "")
			}

			logDir := GetLogDir()
			err = os.MkdirAll(logDir, _const.DefaultNewFilePermission) // create .nhctl/logs
			if err != nil {
				return errors.Wrap(err, "")
			}

		}
	}
	return err
}

func CleanupAppFilesUnderNs(namespace, nid string) error {
	appDir := nocalhost_path.GetNidDir(namespace, nid)
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
	return filepath.Join(nocalhost_path.GetNhctlHomeDir(), _const.DefaultBinDirName, _const.DefaultBinSyncThingDirName)
}

func GetLogDir() string {
	return filepath.Join(nocalhost_path.GetNhctlHomeDir(), _const.DefaultLogDirName)
}

type AppInfo struct {
	Name      string
	Namespace string
	Nid       string
}

// MoveAppFromNsToNid For compatibility
func MoveAppFromNsToNid() error {
	nsBaseDir := nocalhost_path.GetNhctlNameSpaceBaseDir()
	nsList, err := ioutil.ReadDir(nsBaseDir)
	if err != nil {
		return errors.Wrap(err, "")
	}
	wg := sync.WaitGroup{}
	for _, ns := range nsList {
		if !ns.IsDir() {
			continue
		}
		appList, err := ioutil.ReadDir(filepath.Join(nsBaseDir, ns.Name()))
		if err != nil {
			continue
		}
		for _, a := range appList {
			oldAppDir := filepath.Join(nsBaseDir, ns.Name(), a.Name())
			if !IsNocalhostAppDir(oldAppDir) {
				continue
			}
			wg.Add(1)
			go func(a fs.FileInfo, ns fs.FileInfo) {
				defer wg.Done()
				log.Logf("Move %s-%s to nid dir", a.Name(), ns.Name())
				kube, err := GetKubeConfigFromProfile(ns.Name(), a.Name(), "")
				if err != nil {
					log.Logf("Moving %s-%s pass: %s , get kubeconfig failed", a.Name(), ns.Name(), err.Error())
					return
				}
				meta, err := GetApplicationMeta(a.Name(), ns.Name(), kube)
				if err != nil {
					log.Logf("Moving %s-%s pass: %s ", a.Name(), ns.Name(), err.Error())
					return
				}
				if !meta.IsInstalled() || meta.GenerateNidINE() != nil {
					log.Logf("Moving %s-%s pass: %s ", a.Name(), ns.Name(), "meta is not installed")
					return
				}
				if err = MigrateNsDirToSupportNidIfNeeded(a.Name(), ns.Name(), meta.NamespaceId); err != nil {
					log.Logf("Moving %s-%s pass: %s ", a.Name(), ns.Name(), err.Error())
				} else {
					log.Logf("Success to move %s-%s to nid %s dir", a.Name(), ns.Name(), meta.NamespaceId)
				}
			}(a, ns)
		}
	}
	wg.Wait()
	return nil
}

func MigrateNsDirToSupportNidIfNeeded(app, ns, nid string) error {

	newDir := nocalhost_path.GetAppDirUnderNs(app, ns, nid)
	_, err := os.Stat(newDir)
	if os.IsNotExist(err) {
		oldDir := nocalhost_path.GetAppDirUnderNsWithoutNid(app, ns)
		ss, err := os.Stat(oldDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return errors.Wrap(err, "Old Dir occurs errors")
		}
		if !ss.IsDir() {
			return nil
		}

		markedFileName := strings.Join([]string{app, ns, nid, "migrating"}, "-")
		markedFilePath := filepath.Join(nocalhost_path.GetNhctlNameSpaceBaseDir(), ns, markedFileName)
		if _, err = os.Stat(markedFilePath); err == nil {
			return errors.New(fmt.Sprintf("Another process is migrating %s-%s-%s", app, ns, nid))
		}
		if _, err = os.Create(markedFilePath); err != nil {
			return errors.Wrap(err, "Failed to create marked file")
		}
		defer func() {
			if err = os.Remove(markedFilePath); err != nil {
				log.LogE(errors.Wrap(err, "Migrating err"))
			}
		}()

		if err = utils.CopyDir(oldDir, newDir); err != nil {
			_ = os.RemoveAll(newDir)
			return errors.Wrap(err, "Migrating err: failed to copy")
		} else {
			log.Logf("app %s in %s has been copied", app, ns)
			if err = os.RemoveAll(oldDir); err != nil {
				return errors.Wrap(err, "Migrating err")
			}
		}
	}
	return nil
}

func GetNsAndApplicationInfo(portForwardFilter, nidMigrate bool) ([]AppInfo, error) {
	if nidMigrate {
		if err := MoveAppFromNsToNid(); err != nil {
			log.LogE(err)
		}
	}

	result := make([]AppInfo, 0)
	nsDir := nocalhost_path.GetNhctlNameSpaceBaseDir()
	nsList, err := ioutil.ReadDir(nsDir)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	for _, ns := range nsList {
		if !ns.IsDir() {
			continue
		}
		nidDirList, err := ioutil.ReadDir(filepath.Join(nsDir, ns.Name()))
		if err != nil {
			log.WarnE(errors.Wrap(err, ""), "Failed to read dir")
			continue
		}
		for _, nidDir := range nidDirList {
			if nidDir.IsDir() {
				nidPath := filepath.Join(nsDir, ns.Name(), nidDir.Name())
				appDirList, err := ioutil.ReadDir(nidPath)
				if err != nil {
					log.LogE(errors.Wrap(err, "Failed to get app dir list"))
					continue
				}
				for _, appDir := range appDirList {
					if !appDir.IsDir() {
						continue
					}
					appPath := filepath.Join(nidPath, appDir.Name())
					if !IsNocalhostAppDir(appPath) {
						continue
					}
					if portForwardFilter {
						if !IsPortForwarding(appPath) {
							continue
						}
					}
					result = append(
						result, AppInfo{
							Name:      appDir.Name(),
							Namespace: ns.Name(),
							Nid:       nidDir.Name(),
						},
					)
				}
			}
		}
	}
	return result, nil
}

// IsNocalhostAppDir Check if a dir is a nocalhost dir
func IsNocalhostAppDir(dir string) bool {
	s, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !s.IsDir() {
		return false
	}
	appDirItems, err := ioutil.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, item := range appDirItems {
		if !item.IsDir() {
			continue
		}
		if item.Name() == "db" {
			return true
		}
	}
	return false
}

func IsPortForwarding(dir string) bool {
	s, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !s.IsDir() {
		return false
	}
	appDirItems, err := ioutil.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, item := range appDirItems {
		if item.Name() == "portforward" {
			return true
		}
	}
	return false
}

func GetApplicationMeta(appName, namespace, kubeConfig string) (*appmeta.ApplicationMeta, error) {
	cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
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
	cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
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
