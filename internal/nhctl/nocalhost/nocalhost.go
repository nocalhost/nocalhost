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
	"io/ioutil"
	"os"
	"path/filepath"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/utils"
)

const (
	DefaultNewFilePermission = 0700
)

type NocalHost struct {
}

func NewNocalHost() (*NocalHost, error) {
	nh := &NocalHost{}
	err := nh.Init()
	if err != nil {
		return nil, err
	}
	return nh, nil
}

func (n *NocalHost) Init() error {
	var err error
	nhctlHomeDir := n.GetHomeDir()
	if _, err = os.Stat(nhctlHomeDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(nhctlHomeDir, DefaultNewFilePermission)
			if err != nil {
				return err
			}

			applicationDir := n.GetAppHomeDir()
			err = os.MkdirAll(applicationDir, DefaultNewFilePermission) // create .nhctl/application
			if err != nil {
				return err
			}

			binDir := n.GetSyncThingBinDir()
			err = os.MkdirAll(binDir, DefaultNewFilePermission) // create .nhctl/bin/syncthing
			if err != nil {
				return err
			}

			logDir := n.GetLogDir()
			err = os.MkdirAll(logDir, DefaultNewFilePermission) // create .nhctl/logs
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (n *NocalHost) GetHomeDir() string {
	return filepath.Join(utils.GetHomePath(), app.DefaultNhctlHomeDirName)
	//return fmt.Sprintf("%s%c%s", utils.GetHomePath(), os.PathSeparator, app.DefaultNhctlHomeDirName)
}

func (n *NocalHost) GetAppHomeDir() string {
	return filepath.Join(n.GetHomeDir(), app.DefaultApplicationDirName)
	//return fmt.Sprintf("%s%c%s", n.GetHomeDir(), os.PathSeparator, app.DefaultApplicationDirName)
}

func (n *NocalHost) GetAppDir(appName string) string {
	return filepath.Join(n.GetAppHomeDir(), appName)
	//return fmt.Sprintf("%s%c%s", n.GetAppHomeDir(), os.PathSeparator, appName)
}

func (n *NocalHost) CleanupAppFiles(appName string) error {
	appDir := n.GetAppDir(appName)
	if f, err := os.Stat(appDir); err == nil {
		if f.IsDir() {
			err = os.RemoveAll(appDir)
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (n *NocalHost) GetSyncThingBinDir() string {
	return filepath.Join(n.GetHomeDir(), app.DefaultBinDirName, app.DefaultBinSyncThingDirName)
}

func (n *NocalHost) GetLogDir() string {
	return fmt.Sprintf("%s%c%s", n.GetHomeDir(), os.PathSeparator, app.DefaultLogDirName)
}

func (n *NocalHost) GetApplicationNames() ([]string, error) {
	appDir := n.GetAppHomeDir()
	fs, err := ioutil.ReadDir(appDir)
	if err != nil {
		return nil, err
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

func (n *NocalHost) CheckIfApplicationExist(appName string) bool {
	apps, err := n.GetApplicationNames()
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

func (n *NocalHost) GetApplication(appName string) (*app.Application, error) {
	return app.NewApplication(appName)
}
