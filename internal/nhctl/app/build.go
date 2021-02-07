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
	"nocalhost/internal/nhctl/envsubst"
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

	//profile, err := NewAppProfile(app.getProfilePath())
	//if err != nil {
	//	return nil, err
	//}
	//app.AppProfile = profile
	err = app.LoadAppProfileV2()
	if err != nil {
		return nil, err
	}

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
	app.AppProfileV2.Namespace = namespace
	app.AppProfileV2.Kubeconfig = kubeconfig

	if flags.GitUrl != "" {
		err = app.downloadResourcesFromGit(flags.GitUrl, flags.GitRef)
		if err != nil {
			log.Debugf("Failed to clone : %s ref: %s", flags.GitUrl, flags.GitRef)
			return nil, err
		}
	}

	// local path of application, copy to nocalhost resource
	if flags.LocalPath != "" {
		err := utils.CopyDir(flags.LocalPath, app.getApplicationDir(name))
		if err != nil {
			return nil, err
		}
	}

	err = app.generateConfig(flags.OuterConfig, flags.Config)
	if err != nil {
		return nil, err
	}

	// app.LoadSvcConfigsToProfile()
	// Load config to profile
	app.AppProfileV2.AppType = app.configV2.ApplicationConfig.Type
	app.AppProfileV2.ResourcePath = app.configV2.ApplicationConfig.ResourcePath
	app.AppProfileV2.IgnoredPath = app.configV2.ApplicationConfig.IgnoredPath
	app.AppProfileV2.PreInstall = app.configV2.ApplicationConfig.PreInstall
	app.AppProfileV2.Env = app.configV2.ApplicationConfig.Env
	app.AppProfileV2.EnvFrom = app.configV2.ApplicationConfig.EnvFrom
	for _, svcConfig := range app.configV2.ApplicationConfig.ServiceConfigs {
		app.loadConfigToSvcProfile(svcConfig.Name, Deployment)
	}

	if flags.AppType != "" {
		app.AppProfileV2.AppType = AppType(flags.AppType)
	}
	if len(flags.ResourcePath) != 0 {
		app.AppProfileV2.ResourcePath = flags.ResourcePath
	}

	return app, app.SaveProfile()
}

// V2
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

	// Check If config version
	configVersion, err := checkConfigVersion(configFile)
	if err != nil {
		return err
	}

	if configVersion == "v1" {
		err = ConvertConfigFileV1ToV2(configFile, a.GetConfigV2Path())
		if err != nil {
			return err
		}
	} else {
		renderedStr, err := envsubst.RenderBytes(rbytes, "")
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(a.GetConfigV2Path(), []byte(renderedStr), 0644) // replace .nocalhost/config.yam with outerConfig in git or config in absolution path
		if err != nil {
			return errors.New(fmt.Sprintf("fail to create configFile : %s", configFile))
		}
	}

	err = a.LoadConfigV2()
	if err != nil {
		return err
	}

	if a.configV2 == nil {
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

	//err = ioutil.WriteFile(a.getProfilePath(), []byte(""), DefaultNewFilePermission)
	err = ioutil.WriteFile(a.getProfileV2Path(), []byte(""), DefaultConfigFilePermission)
	return errors.Wrap(err, "")
}

// svcName use actual name
func (a *Application) loadConfigToSvcProfile(svcName string, svcType SvcType) {
	if a.AppProfileV2.SvcProfile == nil {
		a.AppProfileV2.SvcProfile = make([]*SvcProfileV2, 0)
	}

	svcProfile := &SvcProfileV2{
		ActualName: svcName,
		//Type:       svcType,
	}

	// find svc config
	svcConfig := a.GetSvcConfigV2(svcName)
	if svcConfig == nil && len(a.AppProfileV2.ReleaseName) > 0 {
		if strings.HasPrefix(svcName, fmt.Sprintf("%s-", a.AppProfileV2.ReleaseName)) {
			name := strings.TrimPrefix(svcName, fmt.Sprintf("%s-", a.AppProfileV2.ReleaseName))
			svcConfig = a.GetSvcConfigV2(name) // support releaseName-svcName
		}
	}

	svcProfile.ServiceConfigV2 = svcConfig

	// If svcProfile already exists, updating it
	for index, svc := range a.AppProfileV2.SvcProfile {
		if svc.ActualName == svcName {
			a.AppProfileV2.SvcProfile[index] = svcProfile
			return
		}
	}

	// If svcProfile already exists, create one
	a.AppProfileV2.SvcProfile = append(a.AppProfileV2.SvcProfile, svcProfile)
}
