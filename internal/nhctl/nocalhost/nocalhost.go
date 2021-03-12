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

package nocalhost

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"nocalhost/internal/nhctl/utils"
)

const (
	DefaultNewFilePermission        = 0700
	DefaultNhctlHomeDirName         = ".nh/nhctl"
	DefaultNhctlNameSpaceDirName    = "ns"
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
	nhctlHomeDir := GetNhctlHomeDir()
	if _, err = os.Stat(nhctlHomeDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(nhctlHomeDir, DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			// Create ns dir
			nsDir := GetNhctlNameSpaceDir()
			err = os.MkdirAll(nsDir, DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			//applicationDir := GetAppHomeDir()
			//err = os.MkdirAll(applicationDir, DefaultNewFilePermission) // create .nhctl/application
			//if err != nil {
			//	return errors.Wrap(err, "")
			//}

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
	err = moveApplicationDirToNsDir()
	if err != nil {
		return err
	}
	os.Rename(GetAppHomeDir(), fmt.Sprintf("%s.bak", GetAppHomeDir()))
	return nil
}

func moveApplicationDirToNsDir() error {

	if _, err := os.Stat(GetNhctlNameSpaceDir()); err != nil {
		if os.IsNotExist(err) {
			log.Log("Creating ns home dir...")
			nsDir := GetNhctlNameSpaceDir()
			err = os.MkdirAll(nsDir, DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}
			// Move application to ns dir
			appHomeDir := GetAppHomeDir()
			appDirList, err := ioutil.ReadDir(appHomeDir)
			if err != nil {
				return errors.Wrap(err, "")
			}
			for _, appDirInfo := range appDirList {
				if !appDirInfo.IsDir() {
					continue
				}
				appDir := filepath.Join(appHomeDir, appDirInfo.Name())
				ns := ""
				// Get ns from v2
				bytes, err := ioutil.ReadFile(filepath.Join(appDir, DefaultApplicationProfileV2Path))
				if err == nil {
					log.Logf("Try to get %s's namespace from v2", appDirInfo.Name())
					profileV2 := &profile.AppProfileV2{}
					err = yaml.Unmarshal(bytes, profileV2)
					if err != nil {
						log.WarnE(errors.Wrap(err, ""), "")
					} else {
						ns = profileV2.Namespace
					}
				}
				if ns == "" {
					// Get ns from v1
					log.Logf("Try to get %s's namespace from v1", appDirInfo.Name())
					bytes, err = ioutil.ReadFile(filepath.Join(appDir, DefaultApplicationProfilePath))
					if err != nil {
						log.WarnE(errors.Wrap(err, ""), "")
						continue
					}
					profileV1 := &profile.AppProfile{}
					err = yaml.Unmarshal(bytes, profileV1)
					if err != nil {
						log.WarnE(errors.Wrap(err, ""), "")
						continue
					}
					ns = profileV1.Namespace
				}
				if ns == "" {
					log.Warnf("Fail to get %s's namespace", appDirInfo.Name())
					continue
				}
				// Create ns dir
				log.Logf("Create ns dir %s", ns)
				err = os.MkdirAll(filepath.Join(GetNhctlNameSpaceDir(), ns), DefaultNewFilePermission)
				if err != nil {
					log.WarnE(errors.Wrap(err, ""), "")
					continue
				}
				// Moving dir
				err = utils.CopyDir(appDir, filepath.Join(GetNhctlNameSpaceDir(), ns, appDirInfo.Name()))
				if err != nil {
					log.WarnE(errors.Wrap(err, ""), "")
				}
			}

		} else {
			return errors.Wrap(err, "")
		}
	} else {
		log.Log("No need to move application dir to ns dir")
	}

	return nil
}

func GetNhctlHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName)
}

// .nh/nhctl/ns
func GetNhctlNameSpaceDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultNhctlNameSpaceDirName)
}

// Deprecated
func GetAppHomeDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultApplicationDirName)
}

// Deprecated
func GetAppDir(appName string) string {
	return filepath.Join(GetAppHomeDir(), appName)
}

func GetAppDirUnderNs(appName string, namespace string) string {
	return filepath.Join(GetNhctlNameSpaceDir(), namespace, appName)
}

// Deprecated
func CleanupAppFiles(appName string) error {
	appDir := GetAppDir(appName)
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

func CleanupAppFilesUnderNs(appName string, namespace string) error {
	appDir := GetAppDirUnderNs(appName, namespace)
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
	return filepath.Join(GetNhctlHomeDir(), DefaultBinDirName, DefaultBinSyncThingDirName)
}

func GetLogDir() string {
	return filepath.Join(GetNhctlHomeDir(), DefaultLogDirName)
}

// key: ns, value: app
func GetNsAndApplicationInfo() (map[string][]string, error) {
	result := make(map[string][]string, 0)
	nsDir := GetNhctlNameSpaceDir()
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

// Deprecated
func GetApplicationNames() ([]string, error) {
	appDir := GetAppHomeDir()
	fs, err := ioutil.ReadDir(appDir)
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	app := make([]string, 0)
	//if fs == nil || len(fs) < 1 {
	//	return app, nil
	//}
	for _, file := range fs {
		if file.IsDir() {
			app = append(app, file.Name())
		}
	}
	return app, err
}

func CheckIfApplicationExist(appName string, namespace string) bool {
	appMap, err := GetNsAndApplicationInfo()
	if err != nil || len(appMap) == 0 {
		return false
	}

	for ns, appList := range appMap {
		if ns != namespace {
			continue
		}
		for _, app := range appList {
			if app == appName {
				return true
			}
		}
	}
	return false
}
