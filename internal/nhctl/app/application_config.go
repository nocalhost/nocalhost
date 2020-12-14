package app

import (
	"strconv"
	"time"
)

const (
	DefaultSideCarImage                      = "codingcorp-docker.pkg.coding.net/nocalhost/public/nocalhost-sidecar:syncthing"
	DefaultDevImage                          = "codingcorp-docker.pkg.coding.net/nocalhost/public/minideb:latest"
	DefaultWorkDir                           = "/home/nocalhost-dev"
	DefaultLocalSyncDirName                  = "."
	DefaultResourcesDir                      = "resources"
	DefaultNhctlHomeDirName                  = ".nh/nhctl"
	DefaultBinDirName                        = "bin"
	DefaultLogDirName                        = "logs"
	DefaultSyncLogFileName                   = "sync-port-forward-child-process.log"
	DefaultApplicationSyncPortForwardPidFile = "sync-port-forward.pid"
	DefaultBinSyncThingDirName               = "syncthing"
	DefaultBackGroundPortForwardLogFileName  = "alone-port-forward-child-process.log"
	DefaultApplicationOnlyPortForwardPidFile = "alone-port-forward.pid"
	DefaultApplicationSyncPidFile            = "syncthing.pid"
	DefaultApplicationDirName                = "application"
	DefaultApplicationProfilePath            = ".profile.yaml"
	DefaultApplicationConfigDirName          = ".nocalhost"
	DefaultApplicationConfigName             = "config.yaml"
	DefaultNewFilePermission                 = 0700
	DefaultClientGoTimeOut                   = time.Minute * 5
	// nhctl init
	// TODO when release
	DefaultInitHelmGitRepo             = "https://github.com/nocalhost/nocalhost.git"
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
	DefaultInitWaitDeployment          = "nocalhost-dep"
	// TODO when release
	DefaultInitHelmResourcePath   = "deployments/chart"
	DefaultInitPortForwardTimeOut = time.Minute * 1
	DefaultInitApplicationGithub  = "{\"source\":\"git\",\"install_type\":\"rawManifest\",\"resource_dir\":[\"manifest/templates\"],\"application_name\":\"bookinfo\",\"application_url\":\"https://github.com/nocalhost/bookinfo.git\"}"
	DefaultInitApplicationCODING  = "{\"source\":\"git\",\"install_type\":\"rawManifest\",\"resource_dir\":[\"manifest/templates\"],\"application_name\":\"bookinfo\",\"application_url\":\"https://e.coding.net/codingcorp/nocalhost/bookinfo.git\"}"
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

type ServiceDevOptions struct {
	Name     string   `json:"name" yaml:"name"`
	Type     SvcType  `json:"serviceType" yaml:"serviceType"`
	GitUrl   string   `json:"gitUrl" yaml:"gitUrl"`
	DevImage string   `json:"devContainerImage" yaml:"devContainerImage"`
	WorkDir  string   `json:"workDir" yaml:"workDir"`
	Sync     []string `json:"syncDirs" yaml:"syncDirs"`
	Ignore   []string `json:"ignores" yaml:"ignores"` // TODO Ignore file list
	DevPort  []string `json:"devPorts" yaml:"devPorts"`
	Jobs     []string `json:"dependJobsLabelSelector" yaml:"dependJobsLabelSelector,omitempty"`
	Pods     []string `json:"dependPodsLabelSelector" yaml:"dependPodsLabelSelector,omitempty"`
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
