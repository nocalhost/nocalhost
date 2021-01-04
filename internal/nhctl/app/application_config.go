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
	"strconv"
	"time"
)

const (
	DefaultSideCarImage     = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"
	DefaultDevImage         = "codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:latest"
	DefaultWorkDir          = "/home/nocalhost-dev"
	DefaultLocalSyncDirName = "."
	DefaultResourcesDir     = "resources"
	//DefaultNhctlHomeDirName                  = ".nh/nhctl"
	//DefaultBinDirName                        = "bin"
	//DefaultLogDirName                        = "logs"
	DefaultSyncLogFileName                   = "sync-port-forward-child-process.log"
	DefaultApplicationSyncPortForwardPidFile = "sync-port-forward.pid"
	//DefaultBinSyncThingDirName               = "syncthing"
	DefaultBackGroundPortForwardLogFileName  = "alone-port-forward-child-process.log"
	DefaultApplicationOnlyPortForwardPidFile = "alone-port-forward.pid"
	DefaultApplicationSyncPidFile            = "syncthing.pid"
	//DefaultApplicationDirName                = "application"

	DefaultApplicationProfilePath      = ".profile.yaml" // runtime config
	DefaultApplicationConfigPath       = ".config.yaml"
	DefaultApplicationConfigDirName    = ".nocalhost"
	DefaultConfigNameInGitNocalhostDir = "config.yaml"
	DefaultNewFilePermission           = 0700
	DefaultClientGoTimeOut             = time.Minute * 5

	// nhctl init
	// TODO when release
	DefaultInitHelmGitRepo             = "https://github.com/nocalhost/nocalhost.git"
	DefaultInitHelmCODINGGitRepo       = "https://e.coding.net/codingcorp/nocalhost/nocalhost.git"
	DefaultInitHelmType                = "helmGit"
	DefaultInitWatchDeployment         = "nocalhost-api"
	DefaultInitWatchWebDeployment      = "nocalhost-web"
	DefaultInitNocalhostService        = "nocalhost-web"
	DefaultInitInstallApplicationName  = "nocalhost"
	DefaultInitUserEmail               = "foo@nocalhost.dev"
	DefaultInitMiniKubePortForwardPort = 31219
	DefaultInitPassword                = "123456"
	DefaultInitAdminUserName           = "admin@admin.com"
	DefaultInitAdminPassWord           = "123456"
	DefaultInitName                    = "Nocalhost"
	DefaultInitWaitNameSpace           = "nocalhost-reserved"
	DefaultInitCreateNameSpaceLabels   = "nocalhost-init"
	DefaultInitWaitDeployment          = "nocalhost-dep"
	// TODO when release
	DefaultInitHelmResourcePath   = "deployments/chart"
	DefaultInitPortForwardTimeOut = time.Minute * 1
	DefaultInitApplicationGithub  = "{\"source\":\"git\",\"install_type\":\"rawManifest\",\"resource_dir\":[\"manifest/templates\"],\"application_name\":\"bookinfo\",\"application_url\":\"https://github.com/nocalhost/bookinfo.git\"}"
	DefaultInitApplicationCODING  = "{\"source\":\"git\",\"install_type\":\"rawManifest\",\"resource_dir\":[\"manifest/templates\"],\"application_name\":\"bookinfo\",\"application_url\":\"https://e.coding.net/codingcorp/nocalhost/bookinfo.git\"}"
	// Init Component Version Control, HEAD means build from tag
	DefaultNocalhostMainBranch        = "HEAD"
	DefaultNocalhostDepDockerRegistry = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-dep"

	// file sync
	DefaultNocalhostSideCarName = "nocalhost-sidecar"
)

type NocalHostAppConfig struct {
	PreInstall   []*PreInstallItem    `json:"onPreInstall" yaml:"onPreInstall"`
	SvcConfigs   []*ServiceDevOptions `json:"services" yaml:"services"`
	Name         string               `json:"name" yaml:"name"`
	Type         AppType              `json:"manifestType" yaml:"manifestType"`
	ResourcePath []string             `json:"resourcePath" yaml:"resourcePath"`
	// old-config
	//AppConfig  *AppConfig           `json:"app_config" yaml:"appConfig"`
}

type PreInstallItem struct {
	Path   string `json:"path" yaml:"path"`
	Weight string `json:"weight" yaml:"weight"`
}

type PersistentVolumeDir struct {
	Path     string `json:"path" yaml:"path"`
	Capacity string `json:"capacity,omitempty" yaml:"capacity,omitempty"`
}

type ServiceDevOptions struct {
	Name                  string                 `json:"name" yaml:"name"`
	Type                  SvcType                `json:"serviceType" yaml:"serviceType"`
	GitUrl                string                 `json:"gitUrl" yaml:"gitUrl"`
	DevImage              string                 `json:"devContainerImage" yaml:"devContainerImage"`
	WorkDir               string                 `json:"workDir" yaml:"workDir"`
	Sync                  []string               `json:"syncDirs" yaml:"syncDirs,omitempty"`
	PersistentVolumeDirs  []*PersistentVolumeDir `json:"persistentVolumeDirs" yaml:"persistentVolumeDirs"`
	BuildCommand          []string               `json:"buildCommand,omitempty" yaml:"buildCommand,omitempty"`
	RunCommand            []string               `json:"runCommand,omitempty" yaml:"runCommand,omitempty"`
	DebugCommand          []string               `json:"debugCommand,omitempty" yaml:"debugCommand,omitempty"`
	HotReloadRunCommand   []string               `json:"hotReloadRunCommand,omitempty" yaml:"hotReloadRunCommand,omitempty"`
	HotReloadDebugCommand []string               `json:"hotReloadDebugCommand,omitempty" yaml:"hotReloadDebugCommand,omitempty"`
	DevContainerShell     string                 `json:"devContainerShell" yaml:"devContainerShell"`
	Ignore                []string               `json:"ignores" yaml:"ignores"` // TODO Ignore file list
	DevPort               []string               `json:"devPorts" yaml:"devPorts"`
	Jobs                  []string               `json:"dependJobsLabelSelector" yaml:"dependJobsLabelSelector,omitempty"`
	Pods                  []string               `json:"dependPodsLabelSelector" yaml:"dependPodsLabelSelector,omitempty"`
	SyncedPattern         []string               `json:"syncFilePattern" yaml:"syncFilePattern"`
	IgnoredPattern        []string               `json:"ignoreFilePattern" yaml:"ignoreFilePattern"`
}

type ComparableItems []*PreInstallItem

func (a ComparableItems) Len() int      { return len(a) }
func (a ComparableItems) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ComparableItems) Less(i, j int) bool {
	iW, err := strconv.Atoi(a[i].Weight)
	if err != nil {
		iW = 0
	}

	jW, err := strconv.Atoi(a[j].Weight)
	if err != nil {
		jW = 0
	}
	return iW < jW
}

func (n *NocalHostAppConfig) GetSvcConfig(name string) *ServiceDevOptions {
	if n.SvcConfigs == nil {
		return nil
	}
	for _, svc := range n.SvcConfigs {
		if svc.Name == name {
			return svc
		}
	}
	return nil
}
