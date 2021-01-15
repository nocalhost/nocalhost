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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"nocalhost/internal/nhctl/utils"
)

const (
	DefaultNewFilePermission   = 0700
	DefaultNhctlHomeDirName    = ".nh/nhctl"
	DefaultApplicationDirName  = "application"
	DefaultBinDirName          = "bin"
	DefaultBinSyncThingDirName = "syncthing"
	DefaultLogDirName          = "logs"
	DefaultLogFileName         = "nhctl.log"
)

func Init() error {
	var err error
	nhctlHomeDir := GetHomeDir()
	if _, err = os.Stat(nhctlHomeDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(nhctlHomeDir, DefaultNewFilePermission)
			if err != nil {
				return errors.Wrap(err, "")
			}

			applicationDir := GetAppHomeDir()
			err = os.MkdirAll(applicationDir, DefaultNewFilePermission) // create .nhctl/application
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

	return nil
}

func GetHomeDir() string {
	return filepath.Join(utils.GetHomePath(), DefaultNhctlHomeDirName)
}

func GetAppHomeDir() string {
	return filepath.Join(GetHomeDir(), DefaultApplicationDirName)
}

func GetAppDir(appName string) string {
	return filepath.Join(GetAppHomeDir(), appName)
}

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

func GetSyncThingBinDir() string {
	return filepath.Join(GetHomeDir(), DefaultBinDirName, DefaultBinSyncThingDirName)
}

func GetLogDir() string {
	return filepath.Join(GetHomeDir(), DefaultLogDirName)
}

func GetApplicationNames() ([]string, error) {
	appDir := GetAppHomeDir()
	fs, err := ioutil.ReadDir(appDir)
	if err != nil {
		return nil, errors.Wrap(err, err.Error())
	}
	app := make([]string, 0)
	if fs == nil || len(fs) < 1 {
		return app, nil
	}
	for _, file := range fs {
		if file.IsDir() {
			app = append(app, file.Name())
		}
	}
	return app, err
}

func CheckIfApplicationExist(appName string) bool {
	apps, err := GetApplicationNames()
	if err != nil || apps == nil {
		return false
	}

	for _, app := range apps {
		if app == appName {
			return true
		}
	}

	return false
}
