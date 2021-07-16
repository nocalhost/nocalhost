/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
	"path/filepath"
	"regexp"
	"strings"
)

// BuildApplication When a application is installed, something representing the application will build, including:
// 1. An directory (NhctlAppDir) under $NhctlHomeDir/ns/$NameSpace will be created and initiated
// 2. An config will be created and upload to the secret in the k8s cluster, it may come from an config file under
//   .nocalhost in your git repository or an outer config file in your local file system
// 3. An leveldb will be created under $NhctlAppDir, it will record the status of this application
// build a new application
func BuildApplication(name string, flags *app_flags.InstallFlags, kubeconfig string, namespace string) (
	*Application, error,
) {

	var err error

	app := &Application{
		Name:       name,
		NameSpace:  namespace,
		KubeConfig: kubeconfig,
	}

	if err = app.initDir(); err != nil {
		return nil, err
	}

	app.ResourceTmpDir, _ = ioutil.TempDir("", "")
	if err = os.MkdirAll(app.ResourceTmpDir, DefaultNewFilePermission); err != nil {
		return nil, errors.New("Fail to create tmp dir for install")
	}

	if err = nocalhostDb.CreateApplicationLevelDB(app.NameSpace, app.Name, false); err != nil {
		return nil, err
	}

	if app.client, err = clientgoutils.NewClientGoUtils(kubeconfig, namespace); err != nil {
		return nil, err
	}

	if flags.GitUrl != "" {
		if err = downloadResourcesFromGit(flags.GitUrl, flags.GitRef, app.ResourceTmpDir); err != nil {
			return nil, err
		}
	} else if flags.LocalPath != "" { // local path of application, copy to nocalhost resource

		if err = utils.CopyDir(
			filepath.Join(flags.LocalPath, ".nocalhost"),
			filepath.Join(app.ResourceTmpDir, ".nocalhost"),
		); err != nil {
			return nil, err
		}

		for _, needToCopy := range flags.ResourcePath {
			if err = utils.CopyDir(
				filepath.Join(flags.LocalPath, needToCopy),
				filepath.Join(app.ResourceTmpDir, needToCopy),
			); err != nil {
				return nil, err
			}
		}
	}

	// load nocalhost config from dir
	config, err := app.loadOrGenerateConfig(flags.OuterConfig, flags.Config, flags.ResourcePath, flags.AppType)
	if err != nil {
		return nil, err
	}

	// try to create a new application meta
	appMeta, err := nocalhost.GetApplicationMeta(name, namespace, kubeconfig)
	if err != nil {
		return nil, err
	}

	if appMeta.IsInstalled() {
		return nil, errors.New(fmt.Sprintf("Application %s - namespace %s has already been installed", name, namespace))
	} else if appMeta.IsInstalling() {
		return nil, errors.New(fmt.Sprintf("Application %s - namespace %s is installing", name, namespace))
	}

	app.appMeta = appMeta

	if err = appMeta.Initial(); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Errorf("Application %s in %s has been installed", app.Name, app.NameSpace)
		}
		return nil, err
	}

	appMeta.Config = config
	appMeta.ApplicationType = appmeta.AppType(flags.AppType)
	if err := appMeta.Update(); err != nil {
		return nil, err
	}

	appProfileV2 := generateProfileFromConfig(config)
	appProfileV2.Secreted = true
	appProfileV2.Namespace = namespace
	appProfileV2.Kubeconfig = kubeconfig

	if len(flags.ResourcePath) != 0 {
		appProfileV2.ResourcePath = flags.ResourcePath
	}

	app.AppType = appProfileV2.AppType

	return app, nocalhost.UpdateProfileV2(app.NameSpace, app.Name, appProfileV2)
}

func (a *Application) loadOrGenerateConfig(
	outerConfig, config string, resourcePath []string, appType string,
) (*profile.NocalHostAppConfigV2, error) {
	var nocalhostConfig *profile.NocalHostAppConfigV2
	var err error

	configFilePath := outerConfig
	// Read from .nocalhost
	if configFilePath == "" {
		if _, err := os.Stat(a.getConfigPathInGitResourcesDir(config)); err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrap(err, "")
			}
			// no config.yaml
			renderedConfig := &profile.NocalHostAppConfigV2{
				ConfigProperties: &profile.ConfigProperties{Version: "v2"},
				ApplicationConfig: &profile.ApplicationConfig{
					Name:           a.Name,
					Type:           appType,
					ResourcePath:   resourcePath,
					IgnoredPath:    nil,
					PreInstall:     nil,
					HelmValues:     nil,
					Env:            nil,
					EnvFrom:        profile.EnvFrom{},
					ServiceConfigs: nil,
				},
			}
			nocalhostConfig = renderedConfig
		} else {
			configFilePath = a.getConfigPathInGitResourcesDir(config)
		}
	}

	// config.yaml found
	if configFilePath != "" {
		if nocalhostConfig, err = RenderConfig(configFilePath); err != nil {
			return nil, err
		}
	}

	return nocalhostConfig, nil
}

func updateProfileFromConfig(appProfileV2 *profile.AppProfileV2, config *profile.NocalHostAppConfigV2) {
	appProfileV2.EnvFrom = config.ApplicationConfig.EnvFrom
	appProfileV2.ResourcePath = config.ApplicationConfig.ResourcePath
	appProfileV2.IgnoredPath = config.ApplicationConfig.IgnoredPath
	appProfileV2.PreInstall = config.ApplicationConfig.PreInstall
	appProfileV2.Env = config.ApplicationConfig.Env

	if len(appProfileV2.SvcProfile) == 0 {
		appProfileV2.SvcProfile = make([]*profile.SvcProfileV2, 0)
	}
	for _, svcConfig := range config.ApplicationConfig.ServiceConfigs {
		var f bool
		for _, svcP := range appProfileV2.SvcProfile {
			if svcP.ActualName == svcConfig.Name {
				svcP.ServiceConfigV2 = svcConfig
				f = true
				break
			}
		}
		if !f {
			svcProfile := &profile.SvcProfileV2{
				ActualName:      svcConfig.Name,
				ServiceConfigV2: svcConfig,
			}
			appProfileV2.SvcProfile = append(appProfileV2.SvcProfile, svcProfile)
		}
	}
}

func generateProfileFromConfig(config *profile.NocalHostAppConfigV2) *profile.AppProfileV2 {
	appProfileV2 := &profile.AppProfileV2{}
	if config == nil || config.ApplicationConfig == nil {
		return appProfileV2
	}
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

func RenderConfigForSvc(configFilePath string) ([]*profile.ServiceConfigV2, error) {
	configFile := fp.NewFilePath(configFilePath)

	var envFile *fp.FilePathEnhance
	envFile = configFile.RelOrAbs("../").RelOrAbs(".env")

	if e := envFile.CheckExist(); e != nil {
		log.Logf(
			`Render %s Nocalhost config without env files, you can put your env file in %s`,
			configFile.Abs(), envFile.Abs(),
		)
		envFile = nil
	} else {
		log.Logf("Render %s Nocalhost config with env files %s", configFile.Abs(), envFile.Abs())
	}

	renderedStr, err := envsubst.Render(configFile, envFile)
	if err != nil {
		return nil, err
	}

	var renderedConfig []*profile.ServiceConfigV2
	if err = yaml.Unmarshal([]byte(renderedStr), &renderedConfig); err != nil {
		var singleSvcConfig profile.ServiceConfigV2
		if err = yaml.Unmarshal([]byte(renderedStr), &singleSvcConfig); err == nil {
			if len(singleSvcConfig.ContainerConfigs) > 0 {
				renderedConfig = append(renderedConfig, &singleSvcConfig)
			}
		}
	}
	return renderedConfig, nil
}

// V2
func RenderConfig(configFilePath string) (*profile.NocalHostAppConfigV2, error) {
	configFile := fp.NewFilePath(configFilePath)

	var envFile *fp.FilePathEnhance
	if relPath := gettingRenderEnvFile(configFilePath); relPath != "" {
		envFile = configFile.RelOrAbs("../").RelOrAbs(relPath)

		if e := envFile.CheckExist(); e != nil {
			log.Logf(
				`Render %s Nocalhost config without env files, we found the env file 
				had been configured as %s, but we can not found in %s`,
				configFile.Abs(), relPath, envFile.Abs(),
			)
		} else {
			log.Logf("Render %s Nocalhost config with env files %s", configFile.Abs(), envFile.Abs())
		}
	} else {
		log.Logf(
			"Render %s Nocalhost config without env files, you config your Nocalhost "+
				"configuration such as: \nconfigProperties:\n  envFile: ./envs/env\n  version: v2",
			configFile.Abs(),
		)
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
		v2TmpDir, _ := ioutil.TempDir("", "")
		if err = os.MkdirAll(v2TmpDir, DefaultNewFilePermission); err != nil {
			return nil, errors.Wrap(err, "Fail to create tmp dir")
		}
		defer func() {
			_ = os.RemoveAll(v2TmpDir)
		}()

		v2Path := filepath.Join(v2TmpDir, DefaultApplicationConfigV2Path)
		if err = ConvertConfigFileV1ToV2(configFilePath, v2Path); err != nil {
			return nil, err
		}

		if renderedStr, err = envsubst.Render(fp.NewFilePath(v2Path), envFile); err != nil {
			return nil, err
		}
	}

	// convert un strict yaml to strict yaml
	renderedConfig := &profile.NocalHostAppConfigV2{}
	if err = yaml.Unmarshal([]byte(renderedStr), renderedConfig); err != nil {
		return nil, err
	}

	// remove the duplicate service config (we allow users to define duplicate service and keep the last one)
	if renderedConfig.ApplicationConfig != nil && renderedConfig.ApplicationConfig.ServiceConfigs != nil {
		var maps = make(map[string]int)

		for i, config := range renderedConfig.ApplicationConfig.ServiceConfigs {
			if _, ok := maps[config.Name]; ok {
				log.Log(
					"Duplicate service %s found, Nocalhost will "+
						"keep the last one according to the sequence",
					config.Name,
				)
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
