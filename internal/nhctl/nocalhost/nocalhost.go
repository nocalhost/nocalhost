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
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/flock"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/profile"
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
	return moveApplicationDirToNsDir()
}

func moveApplicationDirToNsDir() error {
	if _, err := os.Stat(nocalhost_path.GetNhctlNameSpaceDir()); err != nil {
		if os.IsNotExist(err) {
			log.Log("Creating ns home dir...")
			nsDir := nocalhost_path.GetNhctlNameSpaceDir()
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
				// Initial ns dir
				log.Logf("Initial ns dir %s", ns)
				err = os.MkdirAll(filepath.Join(nocalhost_path.GetNhctlNameSpaceDir(), ns), DefaultNewFilePermission)
				if err != nil {
					log.WarnE(errors.Wrap(err, ""), "")
					continue
				}
				// Moving dir
				err = utils.CopyDir(appDir, filepath.Join(nocalhost_path.GetNhctlNameSpaceDir(), ns, appDirInfo.Name()))
				if err != nil {
					log.WarnE(errors.Wrap(err, ""), "")
				}
			}

		} else {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

// Deprecated
func GetAppHomeDir() string {
	return filepath.Join(nocalhost_path.GetNhctlHomeDir(), DefaultApplicationDirName)
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

func GetApplicationMetaInstalled(appName, namespace, kubeConfig string) (*appmeta.ApplicationMeta, error) {
	appMeta, err := GetApplicationMeta(appName, namespace, kubeConfig)
	if err != nil {
		return nil, err
	}
	if appMeta.IsInstalling() {
		return nil, errors.New(fmt.Sprintf("Application %s - namespace %s is installing,  you can use 'nhctl uninstall %s -n %s' to uninstall this applications ", appName, namespace, appName, namespace))
	} else if appMeta.IsNotInstall() {
		return nil, errors.New(fmt.Sprintf("Application %s in ns %s is not installed or under installing, or maybe the kubeconfig provided has not permitted to this namespace ", appName, namespace))
	}
	return appMeta, nil
}

func GetApplicationMeta(appName, namespace, kubeConfig string) (*appmeta.ApplicationMeta, error) {
	cli, err := daemon_client.NewDaemonClient(utils.IsSudoUser())
	if err != nil {
		return nil, err
	}

	bys, err := ioutil.ReadFile(kubeConfig)

	if err != nil {
		return nil, errors.Wrap(err, "Error to get ApplicationMeta")
	}

	appMeta, err := cli.SendGetApplicationMetaCommand(namespace, appName, string(bys))
	if err != nil {
		return nil, err
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

	appMetas, err := cli.SendGetApplicationMetasCommand(namespace, string(bys))
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

// todo it's worked by dir, so it may not be very accurate
func EstimateApplicationCounts(namespace string) int {
	appMap, err := GetNsAndApplicationInfo()
	if err != nil || len(appMap) == 0 {
		return 0
	}

	for ns, appList := range appMap {
		if ns != namespace {
			continue
		}
		return len(appList)
	}
	return 0
}

func GetFirstApplication(namespace string) string {
	appMap, err := GetNsAndApplicationInfo()
	if err != nil || len(appMap) == 0 {
		return ""
	}

	for ns, appList := range appMap {
		if ns != namespace {
			continue
		}
		if len(appList) > 0 {
			return appList[0]
		}
	}
	return ""
}

func NsLock(namespace string) *flock.Flock {
	return flock.New(
		filepath.Join(defaultNsLockDir(), fmt.Sprintf("%s.flock", namespace)))
}

func defaultNsLockDir() string {
	p := filepath.Join(nocalhost_path.GetNhctlHomeDir(), "nslock")
	_ = os.Mkdir(p, 0700)
	return p
}
