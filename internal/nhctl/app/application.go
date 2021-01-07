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
	"fmt"
	"io/ioutil"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/pkg/errors"
)

type AppType string

const (
	Helm     AppType = "helmGit"
	HelmRepo AppType = "helmRepo"
	Manifest AppType = "rawManifest"
)

type Application struct {
	Name                     string
	config                   *NocalHostAppConfig //  this should not be nil
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

	//app.client, err = clientgoutils.NewClientGoUtils(app.GetKubeconfig(), DefaultClientGoTimeOut)
	err = app.InitClient(app.GetKubeconfig(), app.GetNamespace())
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
	a.client, err = clientgoutils.NewClientGoUtils(kubeconfig, namespace)
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
		log.Error("fail to save nocalhostApp profile")
	}
	return err
}

func (a *Application) InitDir() error {
	var err error
	err = os.MkdirAll(a.GetHomeDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = os.MkdirAll(a.getGitDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}

	err = os.MkdirAll(a.getConfigDir(), DefaultNewFilePermission)
	if err != nil {
		return errors.Wrap(err, "")
	}
	err = ioutil.WriteFile(a.getProfilePath(), []byte(""), DefaultNewFilePermission)
	return errors.Wrap(err, "")
}

// Load svcConfig to profile while installing
func (a *Application) LoadSvcConfigsToProfile() {
	a.LoadConfig()
	if len(a.config.SvcConfigs) > 0 {
		for _, svcConfig := range a.config.SvcConfigs {
			a.LoadConfigToSvcProfile(svcConfig.Name, Deployment)
		}
	}
}

func (a *Application) InitConfig(outerConfigPath string, configName string) error {

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

	a.AppProfile.AppType = a.config.Type
	a.AppProfile.ResourcePath = a.config.ResourcePath
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
			a.config = config
			return nil
		} else {
			return errors.Wrap(err, "fail to load configs")
		}
	}
	rbytes, err := ioutil.ReadFile(a.GetConfigPath())
	if err != nil {
		return errors.New(fmt.Sprintf("fail to load configFile : %s", a.GetConfigPath()))
	}
	err = yaml.Unmarshal(rbytes, config)
	if err != nil {
		return errors.Wrap(err, err.Error())
	}
	a.config = config
	return nil
}

func (a *Application) SaveConfig() error {
	if a.config != nil {
		bys, err := yaml.Marshal(a.config)
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
		err = ioutil.WriteFile(a.GetConfigPath(), bys, 0644)
		if err != nil {
			return errors.Wrap(err, err.Error())
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
		gitDirName = strs[len(strs)-1] // todo : for default application name
		if len(gitRef) > 0 {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--branch", gitRef, "--depth", "1", gitUrl, a.getGitDir())
		} else {
			_, err = tools.ExecCommand(nil, true, "git", "clone", "--depth", "1", gitUrl, a.getGitDir())
		}
		if err != nil {
			return errors.Wrap(err, err.Error())
		}
	}
	return nil
}

func (a *Application) GetDependencies() []*SvcDependency {
	result := make([]*SvcDependency, 0)

	if a.config == nil {
		return nil
	}

	svcConfigs := a.config.SvcConfigs
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

// Get local path of resource dirs
// If resource path undefined, use git url
func (a *Application) GetResourceDir() []string {
	var resourcePath []string
	if a.AppProfile != nil && len(a.AppProfile.ResourcePath) != 0 {
		for _, path := range a.AppProfile.ResourcePath {
			fullPath := filepath.Join(a.getGitDir(), path)
			resourcePath = append(resourcePath, fullPath)
		}
		return resourcePath
	}
	//if a.config != nil {
	//if len(a.config.ResourcePath) > 0 {
	//	for _, path := range a.config.ResourcePath {
	//		fullPath := filepath.Join(a.getGitDir(), path)
	//		resourcePath = append(resourcePath, fullPath)
	//	}
	//}
	//return resourcePath
	//} else {
	return []string{a.getGitDir()}
	//}
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

func (a *Application) loadInstallManifest() {
	result := make([]string, 0)
	resourcePaths := a.GetResourceDir()
	// TODO if install pass resourceDir, it should be used
	if len(resourcePaths) > 0 {
		for _, eachPath := range resourcePaths {
			files, _, err := a.getYamlFilesAndDirs(eachPath)
			if err != nil {
				log.WarnE(errors.Wrap(err, ""), fmt.Sprintf("Fail to load manifest in %s", eachPath))
				continue
			}

			for _, file := range files {
				if a.ignoredInInstall(file) {
					continue
				}
				if _, err2 := os.Stat(file); err2 != nil {
					log.WarnE(errors.Wrap(err2, ""), fmt.Sprintf("%s can not be installed", file))
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

func (a *Application) uninstallManifestRecursively() error {
	a.loadInstallManifest()

	if len(a.installManifest) > 0 {
		err := a.client.ApplyForDelete(a.installManifest, true)
		if err != nil {
			fmt.Printf("error occurs when cleaning resources: %v\n", err.Error())
			return errors.Wrap(err, err.Error())
		}
	} else {
		log.Warn("nothing need to be uninstalled ??")
	}
	return nil
}

// Path can be a file or a dir
func (a *Application) getYamlFilesAndDirs(path string) ([]string, []string, error) {
	dirs := make([]string, 0)
	files := make([]string, 0)
	var err error
	stat, err := os.Stat(path)
	if err != nil {
		return nil, nil, errors.Wrap(err, "")
	}

	// If path is a file, return it directly
	if !stat.IsDir() {
		return append(files, path), append(dirs, path), nil
	}
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, nil, err
	}

	PthSep := string(os.PathSeparator)

	for _, fi := range dir {
		if fi.IsDir() {
			dirs = append(dirs, path+PthSep+fi.Name())
			fs, ds, err := a.getYamlFilesAndDirs(path + PthSep + fi.Name())
			if err != nil {
				return files, dirs, err
			}
			dirs = append(dirs, ds...)
			files = append(files, fs...)
		} else {
			ok := strings.HasSuffix(fi.Name(), ".yaml")
			if ok {
				files = append(files, path+PthSep+fi.Name())
			} else if strings.HasSuffix(fi.Name(), ".yml") {
				files = append(files, path+PthSep+fi.Name())
			}
		}
	}
	return files, dirs, nil
}

func (a *Application) loadSortedPreInstallManifest() {
	result := make([]string, 0)
	if a.config != nil && a.config.PreInstall != nil {
		sort.Sort(ComparableItems(a.config.PreInstall))
		for _, item := range a.config.PreInstall {
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

	a.loadSortedPreInstallManifest()

	if len(a.sortedPreInstallManifest) > 0 {
		log.Info("Run pre-install...")
		for _, item := range a.sortedPreInstallManifest {
			err := a.client.Create(item, true, false)
			if err != nil {
				log.Warnf("error occurs when install %s : %s\n", item, err.Error())
			}
		}
	}
}

func (a *Application) cleanPreInstall() {
	a.loadSortedPreInstallManifest()
	if len(a.sortedPreInstallManifest) > 0 {
		err := a.client.ApplyForDelete(a.sortedPreInstallManifest, true)
		if err != nil {
			log.Warnf("error occurs when cleaning pre install resources : %s\n", err.Error())
		}
	}
}

func (a *Application) GetNamespace() string {
	return a.AppProfile.Namespace
}

func (a *Application) GetType() AppType {
	//if a.AppProfile != nil && a.AppProfile.AppType != "" {
	return a.AppProfile.AppType
	//}
	//if a.config == nil {
	//	return "", errors.New("config.yaml not found")
	//}
	//if a.config.Type != "" {
	//	return a.config.Type, nil
	//}
}

func (a *Application) GetKubeconfig() string {
	return a.AppProfile.Kubeconfig
}

func (a *Application) GetApplicationSyncDir(deployment string) string {
	dirPath := filepath.Join(a.GetHomeDir(), nocalhost.DefaultBinSyncThingDirName, deployment)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, 0700)
		if err != nil {
			log.Fatalf("fail to create syncthing directory: %s", dirPath)
		}
	}
	return dirPath
}

func (a *Application) GetSvcConfig(svcName string) *ServiceDevOptions {
	a.LoadConfig() // get the latest config
	if a.config == nil {
		return nil
	}
	if a.config.SvcConfigs != nil && len(a.config.SvcConfigs) > 0 {
		for _, config := range a.config.SvcConfigs {
			if config.Name == svcName {
				return config
			}
		}
	}
	return nil
}

func (a *Application) SaveSvcConfig(svcName string, config *ServiceDevOptions) error {
	//err := a.LoadConfig() // load the latest version config
	//if err != nil {
	//	return err
	//}
	//if a.GetSvcConfig(svcName) == nil {
	//	if len(a.Config.SvcConfigs) == 0 {
	//		a.Config.SvcConfigs = make([]*ServiceDevOptions, 0)
	//	}
	//	a.Config.SvcConfigs = append(a.Config.SvcConfigs, config)
	//} else {
	//	for index, svcConfig := range a.Config.SvcConfigs {
	//		if svcConfig.Name == svcName {
	//			a.Config.SvcConfigs[index] = config
	//		}
	//	}
	//}
	//
	//err = a.SaveConfig()
	//if err != nil {
	//	return err
	//}

	svcPro := a.GetSvcProfile(svcName)
	if svcPro != nil {
		svcPro.ServiceDevOptions = config
	}
	fmt.Printf("%+v\n", svcPro.ServiceDevOptions)
	if len(svcPro.ServiceDevOptions.PersistentVolumeDirs) > 0 {
		for _, pvc := range svcPro.ServiceDevOptions.PersistentVolumeDirs {
			fmt.Printf("+%v\n", pvc)
		}
	}
	return a.AppProfile.Save()
}

func (a *Application) GetDefaultWorkDir(svcName string) string {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil && svcProfile.WorkDir != "" {
		return svcProfile.WorkDir
	}
	return DefaultWorkDir
}

func (a *Application) GetPersistentVolumeDirs(svcName string) []*PersistentVolumeDir {
	svcProfile := a.GetSvcProfile(svcName)
	if svcProfile != nil {
		return svcProfile.PersistentVolumeDirs
	}
	return nil
}

func (a *Application) GetDefaultSideCarImage(svcName string) string {
	return DefaultSideCarImage
}

func (a *Application) GetDefaultDevImage(svcName string) string {
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

	dep, err := clientUtils.GetDeployment(svcName)
	if err != nil {
		return err
	}

	rss, err := clientUtils.GetSortedReplicaSetsByDeployment(svcName)
	if err != nil {
		log.WarnE(err, "Failed to get rs list")
		return err
	}

	// Find previous replicaSet
	if len(rss) < 2 {
		log.Warn("No history to roll back")
		return nil
	}

	var r *v1.ReplicaSet
	var originalPodReplicas *int32
	for _, rs := range rss {
		if rs.Annotations == nil {
			continue
		}
		// Mark the original revision
		if rs.Annotations[DevImageRevisionAnnotationKey] == DevImageRevisionAnnotationValue {
			r = rs
			if rs.Annotations[DevImageOriginalPodReplicasAnnotationKey] != "" {
				podReplicas, _ := strconv.Atoi(rs.Annotations[DevImageOriginalPodReplicasAnnotationKey])
				podReplicas32 := int32(podReplicas)
				originalPodReplicas = &podReplicas32
			}
		}
	}
	if r == nil {
		if !reset {
			return errors.New("Failed to find the proper revision to rollback")
		} else {
			r = rss[0]
		}
	}

	dep.Spec.Template = r.Spec.Template
	if originalPodReplicas != nil {
		dep.Spec.Replicas = originalPodReplicas
	}

	spinner := utils.NewSpinner(" Rolling container's revision back...")
	spinner.Start()
	dep, err = clientUtils.UpdateDeployment(dep, metav1.UpdateOptions{}, true)
	spinner.Stop()
	if err != nil {
		coloredoutput.Fail("Failed to roll revision back")
	} else {
		// Wait until workload ready
		err = a.client.WaitDeploymentLatestRevisionToBeReady(svcName)
		if err != nil {
			return err
		} else {
			coloredoutput.Success("Workload has been rollback")
		}
	}

	return err
}

type PortForwardOptions struct {
	Pid         int      `json:"pid" yaml:"pid"`
	DevPort     []string // 8080:8080 or :8080 means random localPort
	RunAsDaemon bool
}

// svcName use actual name
// used in installing
func (a *Application) LoadConfigToSvcProfile(svcName string, svcType SvcType) {
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

func (a *Application) CheckIfSvcExist(name string, svcType SvcType) (bool, error) {
	switch svcType {
	case Deployment:
		//ctx, _ := context.WithTimeout(context.TODO(), DefaultClientGoTimeOut)
		dep, err := a.client.GetDeployment(name)
		if err != nil {
			return false, errors.Wrap(err, "")
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

func (a *Application) CheckConfigFile(file string) error {
	config := &Config{}
	err := yaml.Unmarshal([]byte(file), config)
	if err != nil {
		return errors.New("Application OuterConfig file format error!")
	}
	return config.CheckValid()
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

func (a *Application) GetSvcDescription(svcName string) string {
	desc := ""
	profile := a.GetSvcProfile(svcName)
	if profile != nil {
		bytes, err := yaml.Marshal(profile)
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
					DevPort:                                value.DevPort,
					Developing:                             value.Developing,
					PortForwarded:                          value.PortForwarded,
					Syncing:                                value.Syncing,
					LocalAbsoluteSyncDirFromDevStartPlugin: value.LocalAbsoluteSyncDirFromDevStartPlugin,
					DevPortList:                            value.DevPortList,
					SyncedPatterns:                         value.SyncedPatterns,
					IgnoredPatterns:                        value.IgnoredPatterns,
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
				DevPort:                                svcProfile.DevPort,
				Name:                                   svcProfile.Name,
				Developing:                             svcProfile.Developing,
				PortForwarded:                          svcProfile.PortForwarded,
				Syncing:                                svcProfile.Syncing,
				LocalAbsoluteSyncDirFromDevStartPlugin: svcProfile.LocalAbsoluteSyncDirFromDevStartPlugin,
				DevPortList:                            svcProfile.DevPortList,
				SyncedPatterns:                         svcProfile.SyncedPatterns,
				IgnoredPatterns:                        svcProfile.IgnoredPatterns,
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
//func (a *Application) GetSyncthingPort(svcName string, options *FileSyncOptions) (*FileSyncOptions, error) {
//	svcProfile := a.GetSvcProfile(svcName)
//	if svcProfile == nil {
//		return options, errors.New("get " + svcName + " profile fail, please reinstall application")
//	}
//	options.RemoteSyncthingPort = svcProfile.RemoteSyncthingPort
//	options.RemoteSyncthingGUIPort = svcProfile.RemoteSyncthingGUIPort
//	options.LocalSyncthingPort = svcProfile.LocalSyncthingPort
//	options.LocalSyncthingGUIPort = svcProfile.LocalSyncthingGUIPort
//	return options, nil
//}

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

func (a *Application) GetPodsFromDeployment(deployment string) (*corev1.PodList, error) {
	return a.client.GetPodsFromDeployment(deployment)
}

func (a *Application) WaitAndGetNocalhostDevContainerPod(deployment string) (podName string, err error) {
	checkPodsList, err := a.GetPodsFromDeployment(deployment)
	if err != nil {
		log.Fatalf("get nocalhost dev container fail when file sync err %s", err.Error())
		return "", err
	}
	found := false
	for _, pod := range checkPodsList.Items {
		if pod.Status.Phase == "Running" {
			for _, container := range pod.Spec.Containers {
				if container.Name == DefaultNocalhostSideCarName {
					found = true
					break
				}
			}
			if found {
				podName = pod.Name
				err = nil
				return
			}
		}
	}
	return "", errors.New("dev container not found")
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
	return a.AppProfile.Save()
}

func (a *Application) SetSyncthingPort(svcName string, remotePort, remoteGUIPort, localPort, localGUIPort int) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = remotePort
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = remoteGUIPort
	a.GetSvcProfile(svcName).LocalSyncthingPort = localPort
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = localGUIPort
	return a.AppProfile.Save()
}

func (a *Application) SetSyncthingProfileEndStatus(svcName string) error {
	a.GetSvcProfile(svcName).RemoteSyncthingPort = 0
	a.GetSvcProfile(svcName).RemoteSyncthingGUIPort = 0
	a.GetSvcProfile(svcName).LocalSyncthingPort = 0
	a.GetSvcProfile(svcName).LocalSyncthingGUIPort = 0
	a.GetSvcProfile(svcName).PortForwarded = false
	a.GetSvcProfile(svcName).Syncing = false
	a.GetSvcProfile(svcName).DevPortList = []string{}
	a.GetSvcProfile(svcName).LocalAbsoluteSyncDirFromDevStartPlugin = []string{}
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
		log.Fatalf("Refresh application profile fail")
	}
	a.GetSvcProfile(svcName).Syncing = is
	return a.AppProfile.Save()
}

func (a *Application) CleanupResources() error {
	log.Info("Remove resource files...")
	homeDir := a.GetHomeDir()
	err := os.RemoveAll(homeDir)
	if err != nil {
		return errors.New(fmt.Sprintf("fail to remove resources dir %s\n", homeDir))
	}
	return nil
}
