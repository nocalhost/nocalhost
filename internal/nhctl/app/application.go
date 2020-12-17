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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"nocalhost/internal/nhctl/coloredoutput"
	secret_config "nocalhost/internal/nhctl/syncthing/secret-config"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type AppType string

const (
	Helm     AppType = "helmGit"
	HelmRepo AppType = "helmRepo"
	Manifest AppType = "rawManifest"
)

type Application struct {
	Name                     string
	Config                   *NocalHostAppConfig //  this should not be nil
	NewConfig                *Config
	AppProfile               *AppProfile // runtime info, this will not be nil
	client                   *clientgoutils.ClientGoUtils
	sortedPreInstallManifest []string // for pre install
	installManifest          []string // for install
}

type PluginGetApplication struct {
	Name                    string                 `json:"name" yaml:"name"`
	ReleaseName             string                 `json:"release_name yaml:releaseName"`
	Namespace               string                 `json:"namespace" yaml:"namespace"`
	Kubeconfig              string                 `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string                 `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType                `json:"app_type" yaml:"appType"`
	SvcProfile              []*SvcProfileForPlugin `json:"svc_profile" yaml:"svcProfile"` // this will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool                   `json:"installed" yaml:"installed"`
	ResourcePath            []string               `json:"resource_path" yaml:"resourcePath"`
}

type PluginGetApplicationService struct {
	Name                    string               `json:"name" yaml:"name"`
	ReleaseName             string               `json:"release_name yaml:releaseName"`
	Namespace               string               `json:"namespace" yaml:"namespace"`
	Kubeconfig              string               `json:"kubeconfig" yaml:"kubeconfig,omitempty"`
	DependencyConfigMapName string               `json:"dependency_config_map_name" yaml:"dependencyConfigMapName,omitempty"`
	AppType                 AppType              `json:"app_type" yaml:"appType"`
	SvcProfile              *SvcProfileForPlugin `json:"svc_profile" yaml:"svcProfile"` // this will not be nil after `dev start`, and after `dev start`, application.GetSvcProfile() should not be nil
	Installed               bool                 `json:"installed" yaml:"installed"`
	ResourcePath            []string             `json:"resource_path" yaml:"resourcePath"`
}

type SvcDependency struct {
	Name string   `json:"name" yaml:"name"`
	Type string   `json:"type" yaml:"type"`
	Jobs []string `json:"jobs" yaml:"jobs,omitempty"`
	Pods []string `json:"pods" yaml:"pods,omitempty"`
}

// build a new application
func BuildApplication(name string) (*Application, error) {

	app := &Application{
		Name: name,
	}

	err := app.InitDir()
	if err != nil {
		return nil, err
	}

	profile, err := NewAppProfile(app.getProfilePath())
	if err != nil {
		return nil, err
	}
	app.AppProfile = profile
	return app, nil
}

func NewApplication(name string) (*Application, error) {
	app := &Application{
		Name: name,
	}

	err := app.LoadConfig()
	if err != nil {
		return nil, err
	}

	profile, err := NewAppProfile(app.getProfilePath())
	if err != nil {
		return nil, err
	}
	app.AppProfile = profile

	app.client, err = clientgoutils.NewClientGoUtils(app.GetKubeconfig(), DefaultClientGoTimeOut)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (a *Application) ReadBeforeWriteProfile() error {
	profile, err := NewAppProfile(a.getProfilePath())
	if err != nil {
		return err
	}
	a.AppProfile = profile
	return nil
}

// if namespace is nil, use namespace defined in kubeconfig
func (a *Application) InitClient(kubeconfig string, namespace string) error {
	// check if kubernetes is available
	var err error
	a.client, err = clientgoutils.NewClientGoUtils(kubeconfig, DefaultClientGoTimeOut)
	if err != nil {
		return err
	}
	if namespace == "" {
		namespace, err = a.client.GetDefaultNamespace()
		if err != nil {
			return err
		}
	}

	// save application info
	a.AppProfile.Namespace = namespace
	a.AppProfile.Kubeconfig = kubeconfig
	err = a.AppProfile.Save()
	if err != nil {
		fmt.Println("[error] fail to save nocalhostApp profile")
	}
	return err
}

func (a *Application) InitDir() error {
	var err error
	err = os.MkdirAll(a.GetHomeDir(), DefaultNewFilePermission)
	if err != nil {
		return err
	}

	err = os.MkdirAll(a.getGitDir(), DefaultNewFilePermission)
	if err != nil {
		return err
	}

	err = os.MkdirAll(a.GetConfigDir(), DefaultNewFilePermission)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(a.getProfilePath(), []byte(""), DefaultNewFilePermission)
	return err
}

func (a *Application) InitConfig(outerConfig string) error {
	configFile := outerConfig
	if outerConfig == "" {
		_, err := os.Stat(a.getConfigPathInGitResourcesDir())
		if err == nil {
			configFile = a.getConfigPathInGitResourcesDir()
		}
	}
	if configFile != "" {
		rbytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return errors.New(fmt.Sprintf("fail to load configFile : %s", configFile))
		}
		err = ioutil.WriteFile(a.GetConfigPath(), rbytes, DefaultNewFilePermission)
		if err != nil {
			return errors.New(fmt.Sprintf("fail to create configFile : %s", configFile))
		}
		err = a.LoadConfig()
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) InitProfile(profile *AppProfile) {
	if profile != nil {
		a.AppProfile = profile
	}
}

func (a *Application) LoadConfig() error {
	config := &NocalHostAppConfig{}
	if _, err := os.Stat(a.GetConfigPath()); err != nil {
		if os.IsNotExist(err) {
			a.Config = config
			return nil
		} else {
			return err
		}
	}
	rbytes, err := ioutil.ReadFile(a.GetConfigPath())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigPath()))
	}
	err = yaml.Unmarshal(rbytes, config)
	if err != nil {
		return err
	}
	a.Config = config
	return nil
}

func (a *Application) SaveConfig() error {
	if a.Config != nil {
		bys, err := yaml.Marshal(a.Config)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(a.GetConfigPath(), bys, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) DownloadResourcesFromGit(gitUrl string, gitRef string) error {
	var (
		err        error
		gitDirName string
	)

	if strings.HasPrefix(gitUrl, "https") || strings.HasPrefix(gitUrl, "git") || strings.HasPrefix(gitUrl, "http") {
		if strings.HasSuffix(gitUrl, ".git") {
			gitDirName = gitUrl[:len(gitUrl)-4]
		} else {
			gitDirName = gitUrl
		}
		strs := strings.Split(gitDirName, "/")
		gitDirName = strs[len(strs)-1] // todo : for default application anme
		if len(gitRef) > 0 {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--branch", gitRef, "--depth", "1", gitUrl, a.getGitDir())
		} else {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--depth", "1", gitUrl, a.getGitDir())
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) GetDependencies() []*SvcDependency {
	result := make([]*SvcDependency, 0)

	if a.Config == nil {
		return nil
	}

	svcConfigs := a.Config.SvcConfigs
	if svcConfigs == nil || len(svcConfigs) == 0 {
		return nil
	}

	for _, svcConfig := range svcConfigs {
		if svcConfig.Pods == nil && svcConfig.Jobs == nil {
			continue
		}
		svcDep := &SvcDependency{
			Name: svcConfig.Name,
			Type: string(svcConfig.Type),
			Jobs: svcConfig.Jobs,
			Pods: svcConfig.Pods,
		}
		result = append(result, svcDep)
	}
	return result
}

func (a *Application) IsHelm() bool {
	return a.AppProfile.AppType == Helm || a.AppProfile.AppType == HelmRepo
}

func (a *Application) IsManifest() bool {
	return a.AppProfile.AppType == Manifest
}

func (a *Application) GetResourceDir() []string {
	var resourcePath []string
	if a.AppProfile != nil && len(a.AppProfile.ResourcePath) != 0 {
		for _, path := range a.AppProfile.ResourcePath {
			//fullPath := fmt.Sprintf("%s%c%s", a.getGitDir(), os.PathSeparator, path)
			fullPath := filepath.Join(a.getGitDir(), path)
			resourcePath = append(resourcePath, fullPath)
		}
		return resourcePath
	}
	if a.Config != nil {
		if len(a.Config.ResourcePath) > 0 {
			for _, path := range a.Config.ResourcePath {
				//fullPath := fmt.Sprintf("%s%c%s", a.getGitDir(), os.PathSeparator, path)
				fullPath := filepath.Join(a.getGitDir(), path)
				resourcePath = append(resourcePath, fullPath)
			}
		}
		return resourcePath
	} else {
		return []string{a.getGitDir()}
	}
}

type HelmFlags struct {
	Debug    bool
	Wait     bool
	Set      []string
	Values   string
	Chart    string
	RepoName string
	RepoUrl  string
	Version  string
}

func (a *Application) InstallManifest() error {
	var err error
	a.preInstall()

	// install manifest recursively, don't install pre-install workload again
	err = a.installManifestRecursively()
	return err
}

func (a *Application) loadInstallManifest() {
	result := make([]string, 0)
	resourcePaths := a.GetResourceDir()
	// TODO if install pass resourceDir, it should be used
	if len(resourcePaths) > 0 {
		for _, eachPath := range resourcePaths {
			files, _, err := a.getYamlFilesAndDirs(eachPath)
			if err != nil {
				log.Warnf("fail to load install manifest: %s\n", err.Error())
				return
			}

			for _, file := range files {
				if a.ignoredInInstall(file) {
					continue
				}
				if _, err2 := os.Stat(file); err2 != nil {
					log.Warnf("%s can not be installed : %s", file, err2.Error())
					continue
				}
				result = append(result, file)
			}
		}
	}
	a.installManifest = result
}

// if a file is a preInstall/postInstall, it should be ignored in installing
func (a *Application) ignoredInInstall(manifest string) bool {
	if len(a.sortedPreInstallManifest) > 0 {
		for _, pre := range a.sortedPreInstallManifest {
			if pre == manifest {
				return true
			}
		}
	}
	return false
}

func (a *Application) installManifestRecursively() error {
	a.loadInstallManifest()
	fmt.Printf("installManifest len %d \n", len(a.installManifest))
	if len(a.installManifest) > 0 {
		err := a.client.ApplyForCreate(a.installManifest, a.GetNamespace(), true)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return err
		}
	} else {
		log.Warn("nothing need to be installed ??")
	}
	return nil
}

func (a *Application) uninstallManifestRecursively() error {
	a.loadInstallManifest()

	if len(a.installManifest) > 0 {
		err := a.client.ApplyForDelete(a.installManifest, a.GetNamespace(), true)
		if err != nil {
			fmt.Printf("error occurs when cleaning resources: %v\n", err.Error())
			return err
		}
	} else {
		log.Warn("nothing need to be uninstalled ??")
	}
	return nil
}

func (a *Application) getYamlFilesAndDirs(dirPth string) (files []string, dirs []string, err error) {
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, nil, err
	}

	PthSep := string(os.PathSeparator)

	for _, fi := range dir {
		if fi.IsDir() {
			dirs = append(dirs, dirPth+PthSep+fi.Name())
			fs, ds, err := a.getYamlFilesAndDirs(dirPth + PthSep + fi.Name())
			if err != nil {
				return files, dirs, err
			}
			dirs = append(dirs, ds...)
			files = append(files, fs...)
		} else {
			ok := strings.HasSuffix(fi.Name(), ".yaml")
			if ok {
				files = append(files, dirPth+PthSep+fi.Name())
			} else if strings.HasSuffix(fi.Name(), ".yml") {
				files = append(files, dirPth+PthSep+fi.Name())
			}
		}
	}
	return files, dirs, nil
}

func (a *Application) loadSortedPreInstallManifest() {
	result := make([]string, 0)
	if a.Config != nil && a.Config.PreInstall != nil {
		sort.Sort(ComparableItems(a.Config.PreInstall))
		for _, item := range a.Config.PreInstall {
			itemPath := filepath.Join(a.getGitDir(), item.Path)
			if _, err2 := os.Stat(itemPath); err2 != nil {
				log.Warnf("%s is not a valid pre install manifest : %s\n", itemPath, err2.Error())
				continue
			}
			result = append(result, itemPath)
		}
	}
	a.sortedPreInstallManifest = result
}

func (a *Application) preInstall() {
	fmt.Println("run pre-install....")

	a.loadSortedPreInstallManifest()

	if len(a.sortedPreInstallManifest) > 0 {
		for _, item := range a.sortedPreInstallManifest {
			err := a.client.Create(item, a.GetNamespace(), true, false)
			if err != nil {
				log.Warnf("error occurs when install %s : %s\n", item, err.Error())
			}
		}
	}
	//return files, nil
}

func (a *Application) cleanPreInstall() {
	a.loadSortedPreInstallManifest()
	if len(a.sortedPreInstallManifest) > 0 {
		err := a.client.ApplyForDelete(a.sortedPreInstallManifest, a.GetNamespace(), true)
		if err != nil {
			log.Warnf("error occurs when cleaning pre install resources : %s\n", err.Error())
		}
	}
}

func (a *Application) InstallHelmInRepo(releaseName string, flags *HelmFlags) error {
	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "--namespace", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	chartName := flags.Chart
	installParams := []string{"install", releaseName}
	if flags.Wait {
		installParams = append(installParams, "--wait")
	}
	//if installFlags.HelmRepoUrl
	if flags.RepoUrl != "" {
		installParams = append(installParams, chartName, "--repo", flags.RepoUrl)
	} else if flags.RepoName != "" {
		installParams = append(installParams, fmt.Sprintf("%s/%s", flags.RepoName, chartName))
	}
	if flags.Version != "" {
		installParams = append(installParams, "--version", flags.Version)
	}

	if len(flags.Set) > 0 {
		for _, set := range flags.Set {
			installParams = append(installParams, "--set", set)
		}
	}
	if flags.Values != "" {
		installParams = append(installParams, "-f", flags.Values)
	}
	installParams = append(installParams, "--timeout", "60m")
	installParams = append(installParams, commonParams...)

	fmt.Println("install helm application, this may take several minutes, please waiting...")

	_, err := tools.ExecCommand(nil, true, "helm", installParams...)
	if err != nil {
		return err
	}
	a.AppProfile.ReleaseName = releaseName
	a.AppProfile.Save()
	fmt.Printf(`helm nocalhost app installed, use "helm list -n %s" to get the information of the helm release`+"\n", a.GetNamespace())
	return nil
}

func (a *Application) InstallHelmInGit(releaseName string, flags *HelmFlags) error {
	resourcesPath := a.GetResourceDir()
	if len(resourcesPath) == 0 {
		log.Fatalf("resourcesPath does not define")
	}
	commonParams := make([]string, 0)
	if a.GetNamespace() != "" {
		commonParams = append(commonParams, "-n", a.GetNamespace())
	}
	if a.GetKubeconfig() != "" {
		commonParams = append(commonParams, "--kubeconfig", a.GetKubeconfig())
	}
	if flags.Debug {
		commonParams = append(commonParams, "--debug")
	}

	params := []string{"install", releaseName, resourcesPath[0]}
	if flags.Wait {
		params = append(params, "--wait")
	}
	if len(flags.Set) > 0 {
		for _, set := range flags.Set {
			params = append(params, "--set", set)
		}
	}
	if flags.Values != "" {
		params = append(params, "-f", flags.Values)
	}
	params = append(params, "--timeout", "60m")
	params = append(params, commonParams...)

	fmt.Println("building dependency...")
	depParams := []string{"dependency", "build", resourcesPath[0]}
	depParams = append(depParams, commonParams...)
	_, err := tools.ExecCommand(nil, true, "helm", depParams...)
	if err != nil {
		fmt.Printf("fail to build dependency for helm app, err: %v\n", err)
		return err
	}

	fmt.Println("install helm application, this may take several minutes, please waiting...")
	_, err = tools.ExecCommand(nil, true, "helm", params...)
	if err != nil {
		fmt.Printf("fail to install helm nocalhostApp, err:%v\n", err)
		return err
	}
	a.AppProfile.ReleaseName = releaseName
	a.AppProfile.Save()
	fmt.Printf(`helm application installed, use "helm list -n %s" to get the information of the helm release`+"\n", a.GetNamespace())
	return nil
}

func (a *Application) InstallDepConfigMap(appType AppType) error {
	appDep := a.GetDependencies()
	if appDep != nil {
		var depForYaml = &struct {
			Dependency  []*SvcDependency `json:"dependency" yaml:"dependency"`
			ReleaseName string           `json:"releaseName" yaml:"releaseName"`
		}{
			Dependency: appDep,
		}
		// release name a.Name
		if appType != Manifest {
			depForYaml.ReleaseName = a.Name
		}
		yamlBytes, err := yaml.Marshal(depForYaml)
		if err != nil {
			return err
		}

		dataMap := make(map[string]string, 0)
		dataMap["nocalhost"] = string(yamlBytes)

		configMap := &corev1.ConfigMap{
			Data: dataMap,
		}

		var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")
		rand.Seed(time.Now().UnixNano())
		b := make([]rune, 4)
		for i := range b {
			b[i] = letterRunes[rand.Intn(len(letterRunes))]
		}
		generateName := fmt.Sprintf("nocalhost-depends-do-not-overwrite-%s", string(b))
		configMap.Name = generateName
		if configMap.Labels == nil {
			configMap.Labels = make(map[string]string, 0)
		}
		configMap.Labels["use-for"] = "nocalhost-dep"
		_, err = a.client.ClientSet.CoreV1().ConfigMaps(a.GetNamespace()).Create(context.TODO(), configMap, metav1.CreateOptions{})
		if err != nil {
			fmt.Errorf("fail to create dependency config %s\n", configMap.Name)
			return err
		} else {
			a.AppProfile.DependencyConfigMapName = configMap.Name
			a.AppProfile.Save()
		}
	}
	fmt.Printf("dependency configmap installed\n")
	return nil
}

func (a *Application) GetNamespace() string {
	return a.AppProfile.Namespace
}

func (a *Application) GetType() (AppType, error) {
	if a.AppProfile != nil && a.AppProfile.AppType != "" {
		return a.AppProfile.AppType, nil
	}
	if a.Config == nil {
		return "", errors.New("config.yaml not found")
	}
	if a.Config.Type != "" {
		return a.Config.Type, nil
	}
	return "", errors.New("can not get app type from config.yaml")
}

func (a *Application) GetKubeconfig() string {
	return a.AppProfile.Kubeconfig
}

func (a *Application) GetApplicationSyncDir(deployment string) string {
	dirPath := filepath.Join(a.GetHomeDir(), DefaultBinSyncThingDirName, deployment)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0700)
		if err != nil {
			log.Fatalf("fail to create syncthing directory: %s", dirPath)
		}
	}
	return dirPath
}

//func (a *Application) SavePortForwardInfo(svcName string, localPort int, remotePort int) error {
//	pid := os.Getpid()
//
//	a.GetSvcProfile(svcName).SshPortForward = &PortForwardOptions{
//		//LocalPort:  localPort,
//		//RemotePort: remotePort,
//		Pid: pid,
//	}
//	return a.AppProfile.Save()
//}

//func (a *Application) ListPortForwardPid(svcName string) []int {
//	result := make([]int, 0)
//	profile := a.GetSvcProfile(svcName)
//	if profile == nil || profile.SshPortForward == nil {
//		return result
//	}
//	if profile.SshPortForward.Pid != 0 {
//		result = append(result, profile.SshPortForward.Pid)
//	}
//	return result
//}

//func (a *Application) StopAllPortForward(svcName string) error {
//	pids := a.ListPortForwardPid(svcName)
//	for _, pid := range pids {
//		_, err := tools.ExecCommand(nil, true, "kill", "-1", fmt.Sprintf("%d", pid))
//		if err != nil {
//			fmt.Printf("failed to stop port forward pid %d, err: %v\n", pid, err)
//			return err
//		}
//	}
//	return nil
//}

func (a *Application) GetSvcConfig(svcName string) *ServiceDevOptions {
	a.LoadConfig() // get the latest config
	if a.Config == nil {
		return nil
	}
	if a.Config.SvcConfigs != nil && len(a.Config.SvcConfigs) > 0 {
		for _, config := range a.Config.SvcConfigs {
			if config.Name == svcName {
				return config
			}
		}
	}
	return nil
}

func (a *Application) SaveSvcConfig(svcName string, config *ServiceDevOptions) error {
	err := a.LoadConfig() // load the latest version config
	if err != nil {
		return err
	}
	if a.GetSvcConfig(svcName) == nil {
		if len(a.Config.SvcConfigs) == 0 {
			a.Config.SvcConfigs = make([]*ServiceDevOptions, 0)
		}
		a.Config.SvcConfigs = append(a.Config.SvcConfigs, config)
	} else {
		for index, svcConfig := range a.Config.SvcConfigs {
			if svcConfig.Name == svcName {
				a.Config.SvcConfigs[index] = config
			}
		}
	}

	return a.SaveConfig()
}

func (a *Application) GetDefaultWorkDir(svcName string) string {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.WorkDir != "" {
		return svcProfile.WorkDir
	}
	//config := a.GetSvcConfig(svcName)
	//result := DefaultWorkDir
	//if config != nil && config.WorkDir != "" {
	//	result = config.WorkDir
	//}
	return DefaultWorkDir
}

func (a *Application) GetDefaultSideCarImage(svcName string) string {
	return DefaultSideCarImage
}

//func (a *Application) GetDefaultLocalSyncDirs(svcName string) []string {
//	config := a.GetSvcConfig(svcName)
//	result := []string{DefaultLocalSyncDirName}
//	if config != nil && config.Sync != nil && len(config.Sync) > 0 {
//		result = config.Sync
//	}
//	return result
//}

func (a *Application) GetDefaultDevImage(svcName string) string {
	//config := a.GetSvcConfig(svcName)
	//result := DefaultDevImage
	//if config != nil && config.DevImage != "" {
	//	result = config.DevImage
	//}
	//return result

	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.DevImage != "" {
		return svcProfile.DevImage
	}
	return DefaultDevImage
}

func (a *Application) GetDefaultDevPort(svcName string) []string {
	config := a.GetSvcProfile(svcName)
	if config != nil && len(config.DevPort) > 0 {
		return config.DevPort
	}
	return []string{}
}

func (a *Application) RollBack(ctx context.Context, svcName string, reset bool) error {
	clientUtils := a.client

	dep, err := clientUtils.GetDeployment(ctx, a.GetNamespace(), svcName)
	if err != nil {
		fmt.Printf("failed to get deployment %s , err : %v\n", dep.Name, err)
		return err
	}

	rss, err := clientUtils.GetSortedReplicaSetsByDeployment(ctx, a.GetNamespace(), svcName)
	if err != nil {
		fmt.Printf("failed to get rs list, err:%v\n", err)
		return err
	}
	// find previous replicaSet
	if len(rss) < 2 {
		fmt.Println("no history to roll back")
		return nil
	}

	var r *v1.ReplicaSet
	for _, rs := range rss {
		if rs.Annotations == nil {
			continue
		}
		if rs.Annotations[DevImageFlagAnnotationKey] == DevImageFlagAnnotationValue {
			r = rs
		}
	}
	if r == nil {
		if !reset {
			return errors.New("fail to find the proper revision to rollback")
		} else {
			r = rss[0]
		}
	}

	dep.Spec.Template = r.Spec.Template

	spinner := utils.NewSpinner(" Rolling container's revision back...")
	spinner.Start()
	_, err = clientUtils.UpdateDeployment(ctx, a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	spinner.Stop()
	if err != nil {
		coloredoutput.Fail("Failed to roll revision back")
		//fmt.Println("failed rolling back")
	} else {
		coloredoutput.Success("Container has been rollback")
	}

	return err
}

type PortForwardOptions struct {
	Pid         int      `json:"pid" yaml:"pid"`
	DevPort     []string // 8080:8080 or :8080 means random localPort
	RunAsDaemon bool
}

//func (a *Application) CleanupSshPortForwardInfo(svcName string) error {
//	svcProfile := a.GetSvcProfile(svcName)
//	if svcProfile == nil {
//		return errors.New(fmt.Sprintf("\"%s\" not found", svcName))
//	}
//	svcProfile.SshPortForward = nil
//	return a.AppProfile.Save()
//}

func (a *Application) LoadOrCreateSvcProfile(svcName string, svcType SvcType) {
	if a.AppProfile.SvcProfile == nil {
		a.AppProfile.SvcProfile = make([]*SvcProfile, 0)
	}

	for _, svc := range a.AppProfile.SvcProfile {
		if svc.ActualName == svcName {
			return
		}
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
			svcConfig = a.GetSvcConfig(name)
		}
	}

	svcProfile.ServiceDevOptions = svcConfig

	a.AppProfile.SvcProfile = append(a.AppProfile.SvcProfile, svcProfile)
}

func (a *Application) CheckIfSvcExist(name string, svcType SvcType) (bool, error) {
	switch svcType {
	case Deployment:
		ctx, _ := context.WithTimeout(context.TODO(), DefaultClientGoTimeOut)
		dep, err := a.client.GetDeployment(ctx, a.GetNamespace(), name)
		if err != nil {
			return false, err
		}
		if dep == nil {
			return false, nil
		} else {
			return true, nil
		}
	default:
		return false, errors.New("unsupported svc type")
	}
	return false, nil
}

func (a *Application) CreateSyncThingSecret(syncSecret *corev1.Secret, ops *DevStartOptions) error {
	// check if secret exist
	exist, err := a.client.GetSecret(context.TODO(), ops.Namespace, syncSecret.Name)
	if exist.Name != "" {
		_ = a.client.DeleteSecret(context.TODO(), ops.Namespace, syncSecret.Name)
	}
	_, err = a.client.CreateSecret(context.TODO(), ops.Namespace, syncSecret, metav1.CreateOptions{})
	if err != nil {
		// TODO check configmap first, and end dev should delete that secret
		return err
		//log.Fatalf("create syncthing secret fail, please try to manual delete %s secret first", syncthing.SyncSecretName)
	}
	return nil
}

func (a *Application) ReplaceImage(ctx context.Context, deployment string, ops *DevStartOptions) error {
	deploymentsClient := a.client.GetDeploymentClient(a.GetNamespace())

	// mark current revision for rollback
	rss, err := a.client.GetSortedReplicaSetsByDeployment(ctx, a.GetNamespace(), deployment)
	if err != nil {
		return err
	}
	if rss != nil && len(rss) > 0 {
		rs := rss[len(rss)-1]
		rs.Annotations[DevImageFlagAnnotationKey] = DevImageFlagAnnotationValue
		_, err = a.client.ClientSet.AppsV1().ReplicaSets(a.GetNamespace()).Update(ctx, rs, metav1.UpdateOptions{})
		if err != nil {
			return errors.New("fail to update rs's annotation")
		}
	}

	scale, err := deploymentsClient.GetScale(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return err
	}

	//fmt.Println("developing deployment: " + deployment)
	fmt.Println("scaling replicas to 1")

	if scale.Spec.Replicas > 1 {
		fmt.Printf("deployment %s's replicas is %d now\n", deployment, scale.Spec.Replicas)
		scale.Spec.Replicas = 1
		_, err = deploymentsClient.UpdateScale(ctx, deployment, scale, metav1.UpdateOptions{})
		if err != nil {
			fmt.Println("failed to scale replicas to 1")
		} else {
			time.Sleep(time.Second * 5) // todo check replicas
			fmt.Println("replicas has been scaled to 1")
		}
	} else {
		fmt.Printf("deployment %s's replicas is already 1\n", deployment)
	}

	fmt.Println("Updating development container...")
	dep, err := a.client.GetDeployment(ctx, a.GetNamespace(), deployment)
	if err != nil {
		//fmt.Printf("failed to get deployment %s , err : %v\n", deployment, err)
		return err
	}

	volName := "nocalhost-shared-volume"
	// shared volume
	vol := corev1.Volume{
		Name: volName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	// syncthing secret volume
	syncthingDir := corev1.Volume{
		Name: secret_config.EmptyDir,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	defaultMode := int32(DefaultNewFilePermission)
	syncthingVol := corev1.Volume{
		Name: secret_config.SecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: deployment + "-" + secret_config.SecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  "config.xml",
						Path: "config.xml",
						Mode: &defaultMode,
					},
					{
						Key:  "cert.pem",
						Path: "cert.pem",
						Mode: &defaultMode,
					},
					{
						Key:  "key.pem",
						Path: "key.pem",
						Mode: &defaultMode,
					},
				},
				DefaultMode: &defaultMode,
			},
		},
	}

	if dep.Spec.Template.Spec.Volumes == nil {
		dep.Spec.Template.Spec.Volumes = make([]corev1.Volume, 0)
	}
	dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, vol, syncthingVol, syncthingDir)

	// syncthing volume mount
	syncthingVolHomeDirMount := corev1.VolumeMount{
		Name:      secret_config.EmptyDir,
		MountPath: secret_config.DefaultSyncthingHome,
		SubPath:   "syncthing",
	}

	// syncthing secret volume
	syncthingVolMount := corev1.VolumeMount{
		Name:      secret_config.SecretName,
		MountPath: secret_config.DefaultSyncthingSecretHome,
		ReadOnly:  false,
	}

	// volume mount
	workDir := a.GetDefaultWorkDir(deployment)
	if ops.WorkDir != "" {
		workDir = ops.WorkDir
	}

	volMount := corev1.VolumeMount{
		Name:      volName,
		MountPath: workDir,
	}

	// default : replace the first container
	devImage := a.GetDefaultDevImage(deployment)
	if ops.DevImage != "" {
		devImage = ops.DevImage
	}

	dep.Spec.Template.Spec.Containers[0].Image = devImage
	dep.Spec.Template.Spec.Containers[0].Name = "nocalhost-dev"
	dep.Spec.Template.Spec.Containers[0].Command = []string{"/bin/sh", "-c", "tail -f /dev/null"}
	dep.Spec.Template.Spec.Containers[0].VolumeMounts = append(dep.Spec.Template.Spec.Containers[0].VolumeMounts, volMount)
	// delete users SecurityContext
	dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{}

	// set the entry
	dep.Spec.Template.Spec.Containers[0].WorkingDir = workDir

	// disable readiness probes
	for i := 0; i < len(dep.Spec.Template.Spec.Containers); i++ {
		dep.Spec.Template.Spec.Containers[i].LivenessProbe = nil
		dep.Spec.Template.Spec.Containers[i].ReadinessProbe = nil
		dep.Spec.Template.Spec.Containers[i].StartupProbe = nil
	}

	sideCarImage := a.GetDefaultSideCarImage(deployment)
	if ops.SideCarImage != "" {
		sideCarImage = ops.SideCarImage
	}
	sideCarContainer := corev1.Container{
		Name:       "nocalhost-sidecar",
		Image:      sideCarImage,
		WorkingDir: workDir,
	}
	sideCarContainer.VolumeMounts = append(sideCarContainer.VolumeMounts, volMount, syncthingVolMount, syncthingVolHomeDirMount)

	// over write syncthing command
	sideCarContainer.Command = []string{"/bin/sh", "-c"}
	sideCarContainer.Args = []string{"unset STGUIADDRESS && cp " + secret_config.DefaultSyncthingSecretHome + "/* " + secret_config.DefaultSyncthingHome + "/ && /bin/entrypoint.sh && /bin/syncthing -home /var/syncthing"}
	dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, sideCarContainer)

	_, err = a.client.UpdateDeployment(ctx, a.GetNamespace(), dep, metav1.UpdateOptions{}, true)
	if err != nil {
		fmt.Printf("update develop container failed : %v \n", err)
		return err
	}

	podList, err := a.client.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
	if err != nil {
		fmt.Printf("failed to get pods, err: %v\n", err)
		return err
	}

	log.Debugf("%d pod found", len(podList)) // should be 2

	// wait podList to be ready
	spinner := utils.NewSpinner(" Waiting pod to start...")
	spinner.Start()

wait:
	for {
		podList, err = a.client.ListPodsOfDeployment(a.GetNamespace(), dep.Name)
		if err != nil {
			fmt.Printf("failed to get pods, err: %v\n", err)
			return err
		}
		if len(podList) == 1 {
			pod := podList[0]
			if pod.Status.Phase != corev1.PodRunning {
				spinner.Update(fmt.Sprintf("waiting for pod %s to be Running", pod.Name))
				//log.Debugf("waiting for pod %s to be Running", pod.Name)
				continue
			}
			if len(pod.Spec.Containers) == 0 {
				log.Fatalf("%s has no container ???", pod.Name)
			}

			// make sure all containers are ready and running
			for _, c := range pod.Spec.Containers {
				if !isContainerReadyAndRunning(c.Name, &pod) {
					spinner.Update(fmt.Sprintf("container %s is not ready, waiting...", c.Name))
					//log.Debugf("container %s is not ready, waiting...", c.Name)
					break wait
				}
			}
			spinner.Update("all containers are ready")
			//log.Info("all containers are ready")
			break
		} else {
			spinner.Update(fmt.Sprintf("waiting pod to be replaced..."))
		}
		<-time.NewTimer(time.Second * 1).C
	}
	spinner.Stop()
	coloredoutput.Success("development container has been updated")
	return nil
}

func isContainerReadyAndRunning(containerName string, pod *corev1.Pod) bool {
	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name == containerName && status.Ready && status.State.Running != nil {
			return true
		}
	}
	return false
}

func (a *Application) LoadConfigFile() error {
	if _, err := os.Stat(a.GetConfigPath()); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	rbytes, err := ioutil.ReadFile(a.GetConfigPath())
	if err != nil {
		return errors.New(fmt.Sprintf("failed to load configFile : %s", a.GetConfigPath()))
	}
	config := &Config{}
	err = yaml.Unmarshal(rbytes, config)
	if err != nil {
		return err
	}
	a.NewConfig = config
	return nil
}

func (a *Application) CheckConfigFile(file string) error {
	config := &Config{}
	err := yaml.Unmarshal([]byte(file), config)
	if err != nil {
		return errors.New("Application Config file format error!")
	}
	return config.CheckValid()
}

func (a *Application) SaveConfigFile(file string) error {
	fileByte := []byte(file)
	err := ioutil.WriteFile(a.GetConfigPath(), fileByte, DefaultNewFilePermission)
	return err
}

func (a *Application) GetConfigFile() (string, error) {
	configFile, err := ioutil.ReadFile(a.GetConfigPath())
	if err == nil {
		return string(configFile), err
	}
	return "", err
}

func (a *Application) GetDescription() string {
	desc := ""
	if a.AppProfile != nil {
		bytes, err := yaml.Marshal(a.AppProfile)
		if err == nil {
			desc = string(bytes)
		}
	}
	return desc
}

func (a *Application) GetPluginDescription(service string) string {
	desc := ""
	if a.AppProfile != nil {
		// get all service profile
		if service == "" {
			svcProfileForPlugin := make([]*SvcProfileForPlugin, 0)
			for _, value := range a.AppProfile.SvcProfile {
				rows := &SvcProfileForPlugin{
					Name:                                   value.Name,
					Type:                                   value.Type,
					GitUrl:                                 value.GitUrl,
					DevImage:                               value.DevImage,
					WorkDir:                                value.WorkDir,
					Sync:                                   value.Sync,
					Ignore:                                 value.Ignore,
					DevPort:                                value.DevPort,
					Developing:                             value.Developing,
					PortForwarded:                          value.PortForwarded,
					Syncing:                                value.Syncing,
					LocalAbsoluteSyncDirFromDevStartPlugin: value.LocalAbsoluteSyncDirFromDevStartPlugin,
					DevPortList:                            value.DevPortList,
				}
				svcProfileForPlugin = append(svcProfileForPlugin, rows)
			}
			result := &PluginGetApplication{
				Name:                    a.Name,
				ReleaseName:             a.AppProfile.ReleaseName,
				Namespace:               a.AppProfile.Namespace,
				Kubeconfig:              a.AppProfile.Kubeconfig,
				DependencyConfigMapName: a.AppProfile.DependencyConfigMapName,
				AppType:                 a.AppProfile.AppType,
				Installed:               a.AppProfile.Installed,
				ResourcePath:            a.AppProfile.ResourcePath,
				SvcProfile:              svcProfileForPlugin,
			}
			bytes, err := yaml.Marshal(result)
			if err == nil {
				desc = string(bytes)
			}
			return desc
		}
		if service != "" {

			svcProfile := a.GetSvcProfile(service)
			if svcProfile == nil {
				return desc
			}
			svcProfileForPlugin := &SvcProfileForPlugin{
				Type:                                   svcProfile.Type,
				GitUrl:                                 svcProfile.GitUrl,
				DevImage:                               svcProfile.DevImage,
				WorkDir:                                svcProfile.WorkDir,
				Sync:                                   svcProfile.Sync,
				Ignore:                                 svcProfile.Ignore,
				DevPort:                                svcProfile.DevPort,
				Name:                                   svcProfile.Name,
				Developing:                             svcProfile.Developing,
				PortForwarded:                          svcProfile.PortForwarded,
				Syncing:                                svcProfile.Syncing,
				LocalAbsoluteSyncDirFromDevStartPlugin: svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin,
				DevPortList:                            svcProfile.DevPortList,
			}
			result := &PluginGetApplicationService{
				Name:                    a.Name,
				ReleaseName:             a.AppProfile.ReleaseName,
				Namespace:               a.AppProfile.Namespace,
				Kubeconfig:              a.AppProfile.Kubeconfig,
				DependencyConfigMapName: a.AppProfile.DependencyConfigMapName,
				AppType:                 a.AppProfile.AppType,
				Installed:               a.AppProfile.Installed,
				ResourcePath:            a.AppProfile.ResourcePath,
				SvcProfile:              svcProfileForPlugin,
			}
			bytes, err := yaml.Marshal(result)
			if err == nil {
				desc = string(bytes)
			}
			return desc
		}
	}
	return desc
}

// for background port-forward
func (a *Application) PortForwardInBackGround(deployment, podName, nameSapce string, localPort, remotePort []int) {
	group := len(localPort)
	if len(localPort) != len(remotePort) {
		log.Fatalf("dev port forward fail, please check you devPort in config\n")
	}
	// wait group
	var wg sync.WaitGroup
	wg.Add(group)
	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed. For the port forwarding we just need
	// the output eventually
	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	// managing termination signal from the terminal. As you can see the stopCh
	// gets closed to gracefully handle its termination.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	var addDevPod []string
	//portForwardResultCh := make(chan string, group)
	//var portForwardResult []string
	for key, sLocalPort := range localPort {
		// stopCh control the port forwarding lifecycle. When it gets closed the
		// port forward will terminate
		stopCh := make(chan struct{}, group)
		// readyCh communicate when the port forward is ready to get traffic
		readyCh := make(chan struct{})
		key := key
		sLocalPort := sLocalPort
		devPod := fmt.Sprintf("%d:%d", sLocalPort, remotePort[key])
		addDevPod = append(addDevPod, devPod)
		fmt.Printf("start dev port forward local %d, remote %d \n", sLocalPort, remotePort[key])
		go func() {
			err := a.PortForwardAPod(clientgoutils.PortForwardAPodRequest{
				Pod: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: nameSapce,
					},
				},
				LocalPort: sLocalPort,
				PodPort:   remotePort[key],
				Streams:   stream,
				StopCh:    stopCh,
				ReadyCh:   readyCh,
			})
			//fmt.Print("start send channel")
			if err != nil {
				//portForwardResultCh <- "0"
				fmt.Printf("port-forward in background fail %s\n", err.Error())
				//return
			}
			//portForwardResultCh <- fmt.Sprintf("%d:%d", sLocalPort, remotePort[key])
		}()
		go func(readyCh *chan struct{}) {
			select {
			case <-*readyCh:
				// append status each success port-forward
				_ = a.AppendDevPortForward(deployment, fmt.Sprintf("%d:%d", sLocalPort, remotePort[key]))
				_ = a.SetPortForwardedStatus(deployment, true)
			}
		}(&readyCh)
	}
	fmt.Print("done go routine\n")
	// update profile addDevPod
	// TODO get from channel and set real port-forward status
	//for range localPort {
	//	r := <-portForwardResultCh
	//	portForwardResult = append(portForwardResult, r)
	//}
	//fmt.Printf("portForwardResult %s\n", portForwardResult)

	//_ = a.SetDevPortForward(deployment, portForwardResult)

	// set port forward status
	//if len(portForwardResult) > 0 {
	//	_ = a.SetPortForwardedStatus(deployment, true)
	//}

	for {
		<-sigs
		fmt.Println("stop port forward")
		//close(stopCh)
		wg.Done()
	}
}

// port-forward use
func (a *Application) SetDevPortForward(svcName string, portList []string) error {
	a.GetSvcProfile(svcName).DevPortList = portList
	return a.AppProfile.Save()
}

func (a *Application) AppendDevPortForward(svcName string, portList string) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		log.Fatalf("refresh application profile fail")
	}
	exist := a.GetSvcProfile(svcName).DevPortList
	a.GetSvcProfile(svcName).DevPortList = append(exist, portList)
	return a.AppProfile.Save()
}

func (a *Application) GetDevPortForward(svcName string) []string {
	return a.GetSvcProfile(svcName).DevPortList
}

// for syncthing use
func (a *Application) GetSyncthingPort(svcName string, options *FileSyncOptions) (*FileSyncOptions, error) {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile == nil {
		return options, errors.New("get " + svcName + " profile fail, please reinstall application")
	}
	options.RemoteSyncthingPort = svcProfile.RemoteSyncthingPort
	options.RemoteSyncthingGUIPort = svcProfile.RemoteSyncthingGUIPort
	options.LocalSyncthingPort = svcProfile.LocalSyncthingPort
	options.LocalSyncthingGUIPort = svcProfile.LocalSyncthingGUIPort
	return options, nil
}

func (a *Application) GetMyBinName() string {
	if runtime.GOOS == "windows" {
		return "nhctl.exe"
	}
	return "nhctl"
}

func (a *Application) GetBackgroundSyncPortForwardPid(deployment string, isTrunc bool) (int, string, error) {
	f, err := ioutil.ReadFile(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	if err != nil {
		return 0, a.GetApplicationBackGroundPortForwardPidFile(deployment), err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, a.GetApplicationBackGroundPortForwardPidFile(deployment), err
	}
	if isTrunc {
		_ = a.SetPidFileEmpty(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	}
	return port, a.GetApplicationBackGroundPortForwardPidFile(deployment), nil
}

func (a *Application) GetBackgroundSyncThingPid(deployment string, isTrunc bool) (int, string, error) {
	f, err := ioutil.ReadFile(a.GetApplicationSyncThingPidFile(deployment))
	if err != nil {
		return 0, a.GetApplicationSyncThingPidFile(deployment), err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, a.GetApplicationSyncThingPidFile(deployment), err
	}
	if isTrunc {
		_ = a.SetPidFileEmpty(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	}
	return port, a.GetApplicationSyncThingPidFile(deployment), nil
}

func (a *Application) GetBackgroundOnlyPortForwardPid(deployment string, isTrunc bool) (int, string, error) {
	f, err := ioutil.ReadFile(a.GetApplicationOnlyPortForwardPidFile(deployment))
	if err != nil {
		return 0, a.GetApplicationOnlyPortForwardPidFile(deployment), err
	}
	port, err := strconv.Atoi(string(f))
	if err != nil {
		return 0, a.GetApplicationOnlyPortForwardPidFile(deployment), err
	}
	if isTrunc {
		_ = a.SetPidFileEmpty(a.GetApplicationBackGroundPortForwardPidFile(deployment))
	}
	return port, a.GetApplicationOnlyPortForwardPidFile(deployment), nil
}

func (a *Application) WriteBackgroundSyncPortForwardPidFile(deployment string, pid int) error {
	file, err := os.OpenFile(a.GetApplicationBackGroundPortForwardPidFile(deployment), os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.New("fail open application file sync background port-forward pid file")
	}
	defer file.Close()
	sPid := strconv.Itoa(pid)
	_, err = file.Write([]byte(sPid))
	if err != nil {
		return err
	}
	return nil
}

func (a *Application) GetSyncthingLocalDirFromProfileSaveByDevStart(svcName string, options *DevStartOptions) (*DevStartOptions, error) {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile == nil {
		return options, errors.New("get " + svcName + " profile fail, please reinstall application")
	}
	options.LocalSyncDir = svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin
	return options, nil
}

func (a *Application) GetPodsFromDeployment(ctx context.Context, namespace, deployment string) (*corev1.PodList, error) {
	return a.client.GetPodsFromDeployment(ctx, namespace, deployment)
}

func (a *Application) PortForwardAPod(req clientgoutils.PortForwardAPodRequest) error {
	return a.client.PortForwardAPod(req)
}

// set pid file empty
func (a *Application) SetPidFileEmpty(filePath string) error {
	return os.Remove(filePath)
}

func (a *Application) SetDevEndProfileStatus(svcName string) error {
	a.GetSvcProfile(svcName).Developing = false
	a.GetSvcProfile(svcName).PortForwarded = false
	a.GetSvcProfile(svcName).Syncing = false
	a.GetSvcProfile(svcName).DevPortList = []string{}
	a.GetSvcProfile(svcName).LocalAbsoluteSyncDirFromDevStartPlugin = []string{}
	return a.AppProfile.Save()
}

func (a *Application) SetSyncthingPort(svcName string, remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = remotePort
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = remoteGUIPort
	a.GetSvcProfile(svcName).LocalSyncthingPort = localPort
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = localGUIPort
	return a.AppProfile.Save()
}

func (a *Application) SetRemoteSyncthingPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetRemoteSyncthingGUIPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalSyncthingPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).LocalSyncthingPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalSyncthingGUIPort(svcName string, port int) error {
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = port
	return a.AppProfile.Save()
}

func (a *Application) SetLocalAbsoluteSyncDirFromDevStartPlugin(svcName string, syncDir []string) error {
	a.GetSvcProfile(svcName).LocalAbsoluteSyncDirFromDevStartPlugin = syncDir
	return a.AppProfile.Save()
}

// end syncthing here

func (a *Application) SetDevelopingStatus(svcName string, is bool) error {
	a.GetSvcProfile(svcName).Developing = is
	return a.AppProfile.Save()
}

func (a *Application) SetInstalledStatus(is bool) error {
	a.AppProfile.Installed = is
	return a.AppProfile.Save()
}

func (a *Application) SetAppType(t AppType) error {
	a.AppProfile.AppType = t
	return a.AppProfile.Save()
}

func (a *Application) SetPortForwardedStatus(svcName string, is bool) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		log.Fatalf("refresh application profile fail")
	}
	a.GetSvcProfile(svcName).PortForwarded = is
	return a.AppProfile.Save()
}

func (a *Application) SetSyncingStatus(svcName string, is bool) error {
	err := a.ReadBeforeWriteProfile()
	if err != nil {
		log.Fatalf("refresh application profile fail")
	}
	a.GetSvcProfile(svcName).Syncing = is
	return a.AppProfile.Save()
}

func (a *Application) Uninstall(force bool) error {

	if a.AppProfile.DependencyConfigMapName != "" {
		log.Debugf("delete config map %s\n", a.AppProfile.DependencyConfigMapName)
		err := a.client.DeleteConfigMapByName(a.AppProfile.DependencyConfigMapName, a.AppProfile.Namespace)
		if err != nil && !force {
			return err
		}
		a.AppProfile.DependencyConfigMapName = ""
		a.AppProfile.Save()
	}

	if a.IsHelm() {
		commonParams := make([]string, 0)
		if a.GetNamespace() != "" {
			commonParams = append(commonParams, "--namespace", a.GetNamespace())
		}
		if a.AppProfile.Kubeconfig != "" {
			commonParams = append(commonParams, "--kubeconfig", a.AppProfile.Kubeconfig)
		}
		installParams := []string{"uninstall", a.Name}
		installParams = append(installParams, commonParams...)
		_, err := tools.ExecCommand(nil, true, "helm", installParams...)
		if err != nil && !force {
			return err
		}
	} else if a.IsManifest() {
		//resourceDir := a.GetResourceDir()
		//files, _, err := a.getYamlFilesAndDirs(resourceDir)
		//if err != nil && !force {
		//	return err
		//}
		//err = a.client.ApplyForDelete(files, a.GetNamespace(), true)
		//if err != nil {
		//	return err
		//}
		//end := time.Now()
		//fmt.Printf("installing takes %f seconds\n", end.Sub(start).Seconds())
		a.cleanPreInstall()
		err := a.uninstallManifestRecursively()
		if err != nil {
			return err
		}
	}

	err := a.CleanupResources()
	if err != nil && !force {
		return err
	}

	return nil
}

func (a *Application) CleanupResources() error {
	fmt.Println("remove resource files...")
	homeDir := a.GetHomeDir()
	err := os.RemoveAll(homeDir)
	if err != nil {
		return errors.New(fmt.Sprintf("fail to remove resources dir %s\n", homeDir))
	}
	return nil
}
