/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
	"nocalhost/internal/nhctl/envsubst/parse"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/internal/nhctl/nocalhost"
	nocalhostDb "nocalhost/internal/nhctl/nocalhost/db"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	customyaml3 "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
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

	if err = appMeta.Initial(true); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Logf("Application %s in %s has been installed", app.Name, app.NameSpace)
		}
		return app, err
	}
	app.appMeta = appMeta
	appMeta.ApplicationType = appmeta.AppType(flags.AppType)

	if err = app.initDir(); err != nil {
		return nil, err
	}

	app.ResourceTmpDir, _ = ioutil.TempDir("", "")
	app.shouldClean = true
	if err = os.MkdirAll(app.ResourceTmpDir, DefaultNewFilePermission); err != nil {
		return nil, errors.New("Fail to create tmp dir for install")
	}

	if err = nocalhostDb.CreateApplicationLevelDB(app.NameSpace, app.Name, app.appMeta.NamespaceId, false); err != nil {
		return nil, err
	}

	if app.client, err = clientgoutils.NewClientGoUtils(kubeconfig, namespace); err != nil {
		return nil, err
	}

	if flags.GitUrl != "" {
		if err = downloadResourcesFromGit(flags.GitUrl, flags.GitRef, app.ResourceTmpDir); err != nil {
			return nil, err
		}
	} else if flags.LocalPath != "" {
		app.ResourceTmpDir = flags.LocalPath
		app.shouldClean = false
	}

	// load nocalhost config from dir
	config, err := app.loadOrGenerateConfig(flags.OuterConfig, flags.Config, flags.ResourcePath, flags.AppType)
	if err != nil {
		return nil, err
	}

	if len(flags.ResourcePath) != 0 {
		//if config.ApplicationConfig == nil {
		//	config.ApplicationConfig = &profile.ApplicationConfig{}
		//}
		config.ApplicationConfig.ResourcePath = flags.ResourcePath
	}

	appMeta.Config = config
	appMeta.Config.Migrated = true
	if err := appMeta.Update(); err != nil {
		return nil, err
	}

	appProfileV2 := &profile.AppProfileV2{}
	appProfileV2.AssociateMigrate = true
	appProfileV2.Secreted = true
	appProfileV2.Namespace = namespace
	appProfileV2.Kubeconfig = kubeconfig
	//appProfileV2.ConfigMigrated = true
	appProfileV2.GenerateIdentifierIfNeeded()

	app.Identifier = appProfileV2.Identifier
	app.AppType = appProfileV2.AppType
	return app, nocalhost.UpdateProfileV2(app.NameSpace, app.Name, app.appMeta.NamespaceId, appProfileV2)
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
				ConfigProperties: profile.ConfigProperties{Version: "v2"},
				ApplicationConfig: profile.ApplicationConfig{
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
		if nocalhostConfig, err = RenderConfig(
			envsubst.LocalFileRenderItem{FilePathEnhance: fp.NewFilePath(configFilePath)},
		); err != nil {
			return nil, err
		}
	}

	return nocalhostConfig, nil
}

func RenderConfigForSvc(renderItem envsubst.RenderItem) ([]*profile.ServiceConfigV2, error) {
	renderedStr, err := envsubst.Render(renderItem, nil)
	if err != nil {
		return nil, err
	}

	var renderedConfig []*profile.ServiceConfigV2
	if err = yaml.Unmarshal([]byte(renderedStr), &renderedConfig); err != nil {
		var singleSvcConfig profile.ServiceConfigV2
		if err = yaml.Unmarshal([]byte(renderedStr), &singleSvcConfig); err == nil {
			if len(singleSvcConfig.ContainerConfigs) > 0 || singleSvcConfig.DependLabelSelector != nil {
				renderedConfig = append(renderedConfig, &singleSvcConfig)
			}
		}
	}
	return renderedConfig, nil
}

// V2
func RenderConfig(renderItem envsubst.RenderItem) (*profile.NocalHostAppConfigV2, error) {
	configFileLocation := renderItem.GetLocation()

	var envFile *fp.FilePathEnhance
	var renderedStr string
	var err error

	// means the config is from local files
	if configFileLocation != "" {

		// 1. load local env file if exist
		// 2. try render first time
		// 3. convert v1 to v2 if needed
		// 4. re render
		configFile := fp.NewFilePath(renderItem.GetLocation())

		if relPath := gettingRenderEnvFile(renderItem.GetLocation()); relPath != "" {
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

		renderedStr, err = envsubst.Render(renderItem, envFile)
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
			if err = ConvertConfigFileV1ToV2(configFileLocation, v2Path); err != nil {
				return nil, err
			}

			if renderedStr, err = envsubst.Render(
				envsubst.LocalFileRenderItem{FilePathEnhance: fp.NewFilePath(v2Path)},
				envFile,
			); err != nil {
				return nil, err
			}
		}
	} else {
		renderedStr, err = envsubst.Render(renderItem, envFile)
		if err != nil {
			return nil, err
		}
	}

	// ------
	// Render end, start to unmarshal the config
	// ------

	// convert un strict yaml to strict yaml
	renderedConfig := &profile.NocalHostAppConfigV2{}
	if err := parseNocalhostConfigEnvFile(
		renderedStr, fp.NewFilePath(configFileLocation), func(node *customyaml3.Node) error {
			_ = node.Decode(renderedConfig)

			parseEnvFromIntoEnv(renderedConfig)
			return nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "")
	}

	if os.Getenv("_NOCALHOST_DEBUG_") != "" {
		marshal, _ := customyaml3.Marshal(renderedConfig)
		log.Debug(string(marshal))
	}

	// ------
	// Unmarshal end, remove the duplicate service config (we allow users to define
	// duplicate service and keep the last one)
	// ------

	if renderedConfig.ApplicationConfig.ServiceConfigs != nil {
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

func parseEnvFromIntoEnv(config *profile.NocalHostAppConfigV2) {
	if config != nil {

		arr := make([]*profile.Env, 0)
		for k, v := range getEnvFromEnvFileArr(config.ApplicationConfig.EnvFrom.EnvFile) {
			arr = append(arr, &profile.Env{Name: k, Value: v})
		}

		// env has high priority than env from
		for _, env := range config.ApplicationConfig.Env {
			arr = append(arr, env)
		}

		config.ApplicationConfig.Env = arr

		for _, svcConfig := range config.ApplicationConfig.ServiceConfigs {
			parseEnvFromIntoEnvForSvcConfig(svcConfig)
		}

		config.ApplicationConfig.EnvFrom.EnvFile = nil
	}
}

func parseEnvFromIntoEnvForSvcConfig(svcConfig *profile.ServiceConfigV2) {
	if svcConfig != nil {
		for _, config := range svcConfig.ContainerConfigs {
			if config.Dev != nil && config.Dev.EnvFrom != nil {

				arr := make([]*profile.Env, 0)
				for k, v := range getEnvFromEnvFileArr(config.Dev.EnvFrom.EnvFile) {
					arr = append(arr, &profile.Env{Name: k, Value: v})
				}

				// env has high priority than env from
				for _, env := range config.Dev.Env {
					arr = append(arr, env)
				}

				config.Dev.Env = arr
				config.Dev.EnvFrom = nil
			}

			if config.Install != nil && config.Install.EnvFrom.EnvFile != nil {
				arr := make([]*profile.Env, 0)
				for k, v := range getEnvFromEnvFileArr(config.Install.EnvFrom.EnvFile) {
					arr = append(arr, &profile.Env{Name: k, Value: v})
				}

				// env has high priority than env from
				for _, env := range config.Install.Env {
					arr = append(arr, env)
				}

				config.Install.Env = arr
				config.Install.EnvFrom.EnvFile = nil
			}
		}
	}
}

func getEnvFromEnvFileArr(envFiles []*profile.EnvFile) map[string]string {
	envs := make(map[string]string, 0)
	for _, file := range envFiles {
		mergeMap(envs, fp.NewFilePath(file.Path).ReadEnvFileKV())
	}
	return envs
}

func mergeMap(front, back map[string]string) {
	for k, v := range back {
		front[k] = v
	}
}

func parseNocalhostConfigEnvFile(yAml string, currentPath *fp.FilePathEnhance, nodeConsumer func(*customyaml3.Node) error) error {
	n := customyaml3.Node{}
	if err := customyaml3.Unmarshal([]byte(yAml), &n); err != nil {
		return err
	}

	doParseNode(&n, currentPath, 0, false)

	return nodeConsumer(&n)
}

// we need to maintain the current absPath
func doParseNode(node *customyaml3.Node, currentPath *fp.FilePathEnhance, envFilePathDepth int, nowHit bool) *fp.FilePathEnhance {
	hc := node.HeadComment
	fc := node.FootComment

	if cm := includeComment(hc); cm != nil {
		currentPath = cm
	}

	if nowHit {
		abs := currentPath.RelOrAbs("../").RelOrAbs(node.Value).Abs()
		node.Value = abs
	}

	//    envFile:
	//    - path: /var/folders/15/_sp2z3dd0fb8fstvwym6x9lh0000gn/T/172729671/.nocalhost/config.yaml/dev.env
	//    - path: /var/folders/15/_sp2z3dd0fb8fstvwym6x9lh0000gn/T/172729671/.nocalhost/config.yaml/dev.env
	// 1. when we found the tag named envFile, start to finding tag 'path'
	// 2. `envFile`'s value is an array, so it's Tas is !!seq
	// 3. then we continue finding node with !!map ( path: xxx )
	// 4. last, when we found a node named path, mark hit as true
	// 5. when we get hit, inject path from comment
	if envFilePathDepth == 1 && node.Tag == "!!seq" {
		envFilePathDepth++
	} else if envFilePathDepth == 2 && node.Tag == "!!map" {
		envFilePathDepth++
	} else {
		envFilePathDepth = max(envFilePathDepth-1, 0)
	}

	hit := false
	for _, n := range node.Content {
		currentPath = doParseNode(n, currentPath, envFilePathDepth, hit)

		// when we find a tag named envFile,
		// then we consider next node as flag envFileTag
		if n.Value == "envFile" {
			envFilePathDepth = 1
		}

		hit = envFilePathDepth == 3 && n.Value == "path"
	}

	if cm := includeComment(fc); cm != nil {
		currentPath = cm
	}

	return currentPath
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func includeComment(comment string) *fp.FilePathEnhance {
	//scanner := bufio.NewScanner(strings.NewReader(comment))

	var lastFp *fp.FilePathEnhance
	// only return first line
	for _, text := range strings.Split(comment, "\n") {
		if strings.HasPrefix(text, parse.AbsSign) {
			withoutPrefix := strings.TrimSpace(strings.TrimPrefix(text, parse.AbsSign))
			lastFp = fp.NewFilePath(withoutPrefix)
		}
	}
	return lastFp
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
