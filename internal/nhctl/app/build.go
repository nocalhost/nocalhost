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
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
)

// When a application is installed, something representing the application will build, including:
// 1. An directory (NhctlAppDir) under $NhctlHomeDir will be created and initiated
// 2. An .config.yaml will be created under $NhctlAppDir, it may come from an config file under .nocalhost in your git repository or an outer config file in your local file system
// 3. An .profile.yaml will be created under $NhctlAppDir, it will record the status of this application
// build a new application
func BuildApplication(name string, flags *app_flags.InstallFlags) (*Application, error) {

	app := &Application{
		Name: name,
	}

	err := app.initDir()
	if err != nil {
		return nil, err
	}

	profile, err := NewAppProfile(app.getProfilePath())
	if err != nil {
		return nil, err
	}
	app.AppProfile = profile

	app.SetInstalledStatus(true)

	kubeconfig := flags.KubeConfig
	if kubeconfig == "" { // use default config
		kubeconfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}
	namespace := flags.Namespace

	//err = app.initClient(kubeconfig, namespace)
	app.client, err = clientgoutils.NewClientGoUtils(kubeconfig, namespace)
	if err != nil {
		return nil, err
	}

	// NameSpace may read from kubeconfig
	if namespace == "" {
		namespace, err = app.client.GetDefaultNamespace()
		if err != nil {
			return nil, err
		}
	}
	app.AppProfile.Namespace = namespace
	app.AppProfile.Kubeconfig = kubeconfig

	if flags.GitUrl != "" {
		err = app.downloadResourcesFromGit(flags.GitUrl, flags.GitRef)
		if err != nil {
			log.Debugf("Failed to clone : %s ref: %s", flags.GitUrl, flags.GitRef)
			return nil, err
		}
	}

	err = app.generateConfig(flags.OuterConfig, flags.Config)
	if err != nil {
		return nil, err
	}

	// app.LoadSvcConfigsToProfile()
	// Load config to profile
	app.AppProfile.AppType = app.config.Type
	app.AppProfile.ResourcePath = app.config.ResourcePath
	if len(app.config.SvcConfigs) > 0 {
		for _, svcConfig := range app.config.SvcConfigs {
			app.loadConfigToSvcProfile(svcConfig.Name, Deployment)
		}
	}

	if flags.AppType != "" {
		app.AppProfile.AppType = AppType(flags.AppType)
	}
	if len(flags.ResourcePath) != 0 {
		app.AppProfile.ResourcePath = flags.ResourcePath
	}
	err = app.AppProfile.Save()

	return app, err
}

func (a *Application) generateConfig(outerConfigPath string, configName string) error {

	configFile := outerConfigPath

	// Read from .nocalhost
	if configFile == "" {
		_, err := os.Stat(a.getConfigPathInGitResourcesDir(configName))
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("Nocalhost config %s not found. Please check if there is a file:\"%s\" under .nocalhost directory in your git repository", a.getConfigPathInGitResourcesDir(configName), configName))
			}
			return errors.Wrap(err, "")
		}
		configFile = a.getConfigPathInGitResourcesDir(configName)
	}

	// Generate config.yaml
	// config.yaml may come from .nocalhost in git or a outer config file in local absolute path
	rbytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return errors.New(fmt.Sprintf("fail to load configFile : %s", configFile))
	}
	err = ioutil.WriteFile(a.GetConfigPath(), rbytes, DefaultNewFilePermission) // replace .nocalhost/config.yam with outerConfig in git or config in absolution path
	if err != nil {
		return errors.New(fmt.Sprintf("fail to create configFile : %s", configFile))
	}
	err = a.LoadConfig()
	if err != nil {
		return err
	}

	if a.config == nil {
		return errors.New("Nocalhost config incorrect")
	}
	return nil
}

// Initiate directory layout of a nhctl application
func (a *Application) initDir() error {
	var err error
	err = os.MkdirAll(a.GetHomeDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = os.MkdirAll(a.getGitDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}

	//err = os.MkdirAll(a.getConfigDir(), DefaultNewFilePermission)
	//if err != nil {
	//	return errors.Wrap(err, "")
	//}
	err = ioutil.WriteFile(a.getProfilePath(), []byte(""), DefaultNewFilePermission)
	return errors.Wrap(err, "")
}

// svcName use actual name
func (a *Application) loadConfigToSvcProfile(svcName string, svcType SvcType) {
	if a.AppProfile.SvcProfile == nil {
		a.AppProfile.SvcProfile = make([]*SvcProfile, 0)
	}

	svcProfile := &SvcProfile{
		ActualName: svcName,
		//Type:       svcType,
	}

	// find svc config
	svcConfig := a.GetSvcConfig(svcName)
	if svcConfig == nil && len(a.AppProfile.ReleaseName) > 0 {
		if strings.HasPrefix(svcName, fmt.Sprintf("%s-", a.AppProfile.ReleaseName)) {
			name := strings.TrimPrefix(svcName, fmt.Sprintf("%s-", a.AppProfile.ReleaseName))
			svcConfig = a.GetSvcConfig(name) // support releaseName-svcName
		}
	}

	svcProfile.ServiceDevOptions = svcConfig

	// If svcProfile already exists, updating it
	for index, svc := range a.AppProfile.SvcProfile {
		if svc.ActualName == svcName {
			a.AppProfile.SvcProfile[index] = svcProfile
			return
		}
	}

	// If svcProfile already exists, create one
	a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcProfile)
}
