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
	"bufio"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// When a application is installed, something representing the application will build, including:
// 1. An directory (NhctlAppDir) under $NhctlHomeDir will be created and initiated
// 2. An .config_v2.yaml will be created under $NhctlAppDir, it may come from an config file under .nocalhost in your git repository or an outer config file in your local file system
// 3. An .profile_v2.yaml will be created under $NhctlAppDir, it will record the status of this application
// build a new application
func BuildApplication(name string, flags *app_flags.InstallFlags, kubeconfig string, namespace string) (*Application, error) {

	app := &Application{
		Name:      name,
		NameSpace: namespace,
	}

	err := app.initDir()
	if err != nil {
		return nil, err
	}

	//err = app.LoadAppProfileV2(false)
	//if err != nil {
	//	return nil, err
	//}
	app.AppProfileV2 = &profile.AppProfileV2{}

	app.SetInstalledStatus(true)

	if kubeconfig == "" { // use default config
		kubeconfig = filepath.Join(utils.GetHomePath(), ".kube", "config")
	}

	app.client, err = clientgoutils.NewClientGoUtils(kubeconfig, namespace)
	if err != nil {
		return nil, err
	}

	app.AppProfileV2.Namespace = namespace
	app.AppProfileV2.Kubeconfig = kubeconfig

	if flags.GitUrl != "" {
		err = app.downloadResourcesFromGit(flags.GitUrl, flags.GitRef)
		if err != nil {
			log.Debugf("Failed to clone : %s ref: %s", flags.GitUrl, flags.GitRef)
			return nil, err
		}
		app.AppProfileV2.GitUrl = flags.GitUrl
		app.AppProfileV2.GitUrl = flags.GitRef
	} else if flags.LocalPath != "" { // local path of application, copy to nocalhost resource
		err := utils.CopyDir(flags.LocalPath, app.getGitDir())
		if err != nil {
			return nil, err
		}
	}

	err = app.renderConfig(flags.OuterConfig, flags.Config)
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
		app.AppProfileV2.AppType = flags.AppType
	}
	if len(flags.ResourcePath) != 0 {
		app.AppProfileV2.ResourcePath = flags.ResourcePath
	}

	return app, app.SaveProfile()
}

// V2
func (a *Application) renderConfig(outerConfigPath string, configName string) error {
	configFilePath := outerConfigPath

	// Read from .nocalhost
	if configFilePath == "" {
		_, err := os.Stat(a.getConfigPathInGitResourcesDir(configName))
		if err != nil {
			if os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("Nocalhost config %s not found. Please check if there is a file:\"%s\" under .nocalhost directory in your git repository", a.getConfigPathInGitResourcesDir(configName), configName))
			}
			return errors.Wrap(err, "")
		}
		configFilePath = a.getConfigPathInGitResourcesDir(configName)
	}

	configFile := fp.NewFilePath(configFilePath)

	var envFile *fp.FilePathEnhance
	if relPath := gettingRenderEnvFile(configFilePath); relPath != "" {
		envFile = configFile.RelOrAbs("../").RelOrAbs(relPath)

		if e := envFile.CheckExist(); e != nil {
			log.Log("Render %s Nocalhost config without env files, we found the env file had been configured as %s, but we can not found in %s", configFile.Abs(), relPath, envFile.Abs())
		} else {
			log.Log("Render %s Nocalhost config with env files %s", configFile.Abs(), envFile.Abs())
		}
	} else {
		log.Log("Render %s Nocalhost config without env files, you config your Nocalhost configuration such as: \nconfigProperties:\n  envFile: ./envs/env\n  version: v2", configFile.Abs())
	}

	renderedStr, err := envsubst.Render(configFile, envFile)
	if err != nil {
		return err
	}

	// Check If config version
	configVersion, err := checkConfigVersion(renderedStr)
	if err != nil {
		return err
	}

	if configVersion == "v1" {
		err = ConvertConfigFileV1ToV2(configFilePath, a.GetConfigV2Path())
		if err != nil {
			return err
		}

		renderedStr, err = envsubst.Render(fp.NewFilePath(a.GetConfigV2Path()), envFile)
	}

	// convert un strict yaml to strict yaml
	renderedConfig := &profile.NocalHostAppConfigV2{}
	_ = yaml.Unmarshal([]byte(renderedStr), renderedConfig)

	// remove the duplicate service config (we allow users to define duplicate service and keep the last one)
	if renderedConfig.ApplicationConfig != nil && renderedConfig.ApplicationConfig.ServiceConfigs != nil {
		var maps = make(map[string]int)

		for i, config := range renderedConfig.ApplicationConfig.ServiceConfigs {
			if _, ok := maps[config.Name]; ok {
				log.Log("Duplicate service %s found, Nocalhost will keep the last one according to the sequence", config.Name)
			}
			maps[config.Name] = i
		}

		var service []*profile.ServiceConfigV2
		for _, i := range maps {
			service = append(service, renderedConfig.ApplicationConfig.ServiceConfigs[i])
		}

		renderedConfig.ApplicationConfig.ServiceConfigs = service
	}

	marshal, _ := yaml.Marshal(renderedConfig)

	err = ioutil.WriteFile(a.GetConfigV2Path(), marshal, 0644) // replace .nocalhost/config.yam with outerConfig in git or config in absolution path
	if err != nil {
		return errors.New(fmt.Sprintf("fail to create configFile : %s", configFilePath))
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

func gettingRenderEnvFile(filepath string) string {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	startMatch := false
	for scanner.Scan() {
		text := scanner.Text()
		pureText := strings.TrimSpace(text)

		// disgusting but working
		if strings.HasPrefix(text, "configProperties:") {
			startMatch = true
		} else if startMatch && strings.HasPrefix(text, " ") {

			if strings.HasPrefix(pureText, "envFile: ") {
				value := strings.TrimSpace(text[11:])

				reg := regexp.MustCompile(`^["'](.*)["']$`)
				result := reg.FindAllStringSubmatch(value, -1)

				if len(result) > 0 && len(result[0]) > 1 {
					return result[0][1]
				} else {
					// return the origin value if not matched
					return value
				}
			} else {
				// ignore other node under `configProperties`
			}

		} else if pureText == "" {
			// skip empty line
			continue
		} else if strings.HasPrefix(pureText, "#") {
			// skip comment
			continue
		} else {
			// reset matching
			startMatch = false
		}
	}

	return ""
}

// Initiate directory layout of a nhctl application
func (a *Application) initDir() error {
	var err error
	if err = os.MkdirAll(a.GetHomeDir(), DefaultNewFilePermission); err != nil {
		return errors.Wrap(err, "")
	}

	if err = os.MkdirAll(a.getGitDir(), DefaultNewFilePermission); err != nil {
		return errors.Wrap(err, "")
	}

	log.Infof("Making dir %s", a.getDbDir())
	return errors.Wrap(os.MkdirAll(a.getDbDir(), DefaultNewFilePermission), "")
}

// svcName use actual name
func (a *Application) loadConfigToSvcProfile(svcName string, svcType SvcType) {
	if a.AppProfileV2.SvcProfile == nil {
		a.AppProfileV2.SvcProfile = make([]*profile.SvcProfileV2, 0)
	}

	svcProfile := &profile.SvcProfileV2{
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
