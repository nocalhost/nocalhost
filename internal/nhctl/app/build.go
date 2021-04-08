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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/envsubst"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"regexp"
	"strings"
)

// When a application is installed, something representing the application will build, including:
// 1. An directory (NhctlAppDir) under $NhctlHomeDir/ns/$NameSpace will be created and initiated
// 2. An .config_v2.yaml will be created under $NhctlAppDir, it may come from an config file under .nocalhost in your git repository or an outer config file in your local file system
// 3. An .profile_v2.yaml will be created under $NhctlAppDir, it will record the status of this application
// build a new application
func BuildApplication(name string, flags *app_flags.InstallFlags, kubeconfig string, namespace string) (*Application, error) {

	app := &Application{
		Name:       name,
		NameSpace:  namespace,
		KubeConfig: kubeconfig,
	}

	err := app.initDir()
	if err != nil {
		return nil, err
	}

	app.ResourceTmpDir, _ = ioutil.TempDir("", "")
	err = os.MkdirAll(app.ResourceTmpDir, DefaultNewFilePermission)
	if err != nil {
		return nil, errors.New("Fail to create tmp dir for install")
	}

	if err = nocalhostDb.CreateApplicationLevelDB(app.NameSpace, app.Name, false); err != nil {
		return nil, err
	}

	if app.client, err = clientgoutils.NewClientGoUtils(kubeconfig, namespace); err != nil {
		return nil, err
	}

	if flags.GitUrl != "" {
		if err = app.downloadResourcesFromGit(flags.GitUrl, flags.GitRef, app.ResourceTmpDir); err != nil {
			log.Debugf("Failed to clone : %s ref: %s", flags.GitUrl, flags.GitRef)
			return nil, err
		}
	} else if flags.LocalPath != "" { // local path of application, copy to nocalhost resource
		if err = utils.CopyDir(flags.LocalPath, app.ResourceTmpDir); err != nil {
			return nil, err
		}
	}

	var config *profile.NocalHostAppConfigV2

	configFilePath := flags.OuterConfig
	// Read from .nocalhost
	if configFilePath == "" {
		_, err := os.Stat(app.getConfigPathInGitResourcesDir(flags.Config))
		if err != nil {
			if os.IsNotExist(err) {
				// no config.yaml
				renderedConfig := &profile.NocalHostAppConfigV2{
					ConfigProperties: &profile.ConfigProperties{Version: "v2"},
					ApplicationConfig: &profile.ApplicationConfig{
						Name:           name,
						Type:           flags.AppType,
						ResourcePath:   flags.ResourcePath,
						IgnoredPath:    nil,
						PreInstall:     nil,
						HelmValues:     nil,
						Env:            nil,
						EnvFrom:        profile.EnvFrom{},
						ServiceConfigs: nil,
					},
				}
				configBys, err := yaml.Marshal(renderedConfig)
				if err = ioutil.WriteFile(app.GetConfigV2Path(), configBys, 0644); err != nil {
					return nil, errors.New("fail to create configFile")
				}
				config = renderedConfig
			} else {
				return nil, errors.Wrap(err, "")
			}
		} else {
			configFilePath = app.getConfigPathInGitResourcesDir(flags.Config)
		}
	}

	// config.yaml found
	if configFilePath != "" {
		config, err = app.renderConfig(configFilePath)
		if err != nil {
			return nil, err
		}
	}

	// try to create a new application meta
	appMeta, err := nocalhost.GetApplicationMeta(name, namespace, kubeconfig)
	if err != nil {
		return nil, err
	}

	if appMeta.IsInstalled() {
		return nil, errors.New(fmt.Sprintf("Application %s - namespace %s has already been installed,  you can use 'nhctl uninstall %s -n %s' to uninstall this applications ", name, namespace, name, namespace))
	} else if appMeta.IsInstalling() {
		return nil, errors.New(fmt.Sprintf("Application %s - namespace %s is installing,  you can use 'nhctl uninstall %s -n %s' to uninstall this applications ", name, namespace, name, namespace))
	}

	app.appMeta = appMeta

	if err = appMeta.Initial(); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Application %s has been installed, you can use 'nhctl uninstall %s -n %s' to uninstall this applications ", app.Name, app.Name, app.NameSpace)
		} else {
			return nil, err
		}
	}

	appMeta.Config = config
	appMeta.ApplicationType = appmeta.AppType(flags.AppType)
	if err := appMeta.Update(); err != nil {
		return nil, err
	}

	appProfileV2 := generateProfileFromConfig(config)
	appProfileV2.Namespace = namespace
	appProfileV2.Kubeconfig = kubeconfig

	if len(flags.ResourcePath) != 0 {
		appProfileV2.ResourcePath = flags.ResourcePath
	}

	app.profileV2 = appProfileV2

	return app, nocalhost.UpdateProfileV2(app.NameSpace, app.Name, appProfileV2)
}

func generateProfileFromConfig(config *profile.NocalHostAppConfigV2) *profile.AppProfileV2 {
	appProfileV2 := &profile.AppProfileV2{}
	appProfileV2.EnvFrom = config.ApplicationConfig.EnvFrom
	appProfileV2.ResourcePath = config.ApplicationConfig.ResourcePath
	appProfileV2.IgnoredPath = config.ApplicationConfig.IgnoredPath
	appProfileV2.PreInstall = config.ApplicationConfig.PreInstall
	appProfileV2.Env = config.ApplicationConfig.Env

	appProfileV2.SvcProfile = make([]*profile.SvcProfileV2, 0)
	for _, svcConfig := range config.ApplicationConfig.ServiceConfigs {
		svcProfile := &profile.SvcProfileV2{
			ActualName: svcConfig.Name,
		}
		svcProfile.ServiceConfigV2 = svcConfig
		appProfileV2.SvcProfile = append(appProfileV2.SvcProfile, svcProfile)
	}
	return appProfileV2
}

// V2
func (a *Application) renderConfig(configFilePath string) (*profile.NocalHostAppConfigV2, error) {
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
		return nil, err
	}

	// Check If config version
	configVersion, err := checkConfigVersion(renderedStr)
	if err != nil {
		return nil, err
	}

	if configVersion == "v1" {
		err = ConvertConfigFileV1ToV2(configFilePath, a.GetConfigV2Path())
		if err != nil {
			return nil, err
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

	return renderedConfig, nil
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

	return errors.Wrap(os.MkdirAll(a.getDbDir(), DefaultNewFilePermission), "")
}

// svcName use actual name
func (a *Application) loadConfigToSvcProfile(svcName string, appProfile *profile.AppProfileV2, svcType SvcType) {
	if appProfile.SvcProfile == nil {
		appProfile.SvcProfile = make([]*profile.SvcProfileV2, 0)
	}

	svcProfile := &profile.SvcProfileV2{
		ActualName: svcName,
		//Type:       svcType,
	}

	// find svc config
	svcConfig := a.appMeta.Config.GetSvcConfigV2(svcName)
	if svcConfig == nil && len(appProfile.ReleaseName) > 0 {
		if strings.HasPrefix(svcName, fmt.Sprintf("%s-", appProfile.ReleaseName)) {
			name := strings.TrimPrefix(svcName, fmt.Sprintf("%s-", appProfile.ReleaseName))
			svcConfig = a.appMeta.Config.GetSvcConfigV2(name) // support releaseName-svcName
		}
	}

	svcProfile.ServiceConfigV2 = svcConfig

	// If svcProfile already exists, updating it
	//for index, svc := range appProfile.SvcProfile {
	//	if svc.ActualName == svcName {
	//		appProfile.SvcProfile[index] = svcProfile
	//		return
	//	}
	//}

	// If svcProfile already exists, create one
	appProfile.SvcProfile = append(appProfile.SvcProfile, svcProfile)
}
